package downloader

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type DownloadChunk struct {
	Offset int64
	Start  int64
}

type ResumeDownload struct {
	DownloadChunks []DownloadChunk
}

type DownloadRequest struct {
	Url            string
	FileName       string
	threads        int
	RetryCount     int
	StreamDownload bool
	CustomHeaders  map[string]string
}
type ResponseInfo struct {
	DownloadSpeed  float64
	TotalSize      int64
	DownloadedSize int64
}

func (r *ResponseInfo) GetDownloadedSize() int64  { return r.DownloadedSize }
func (r *ResponseInfo) GetTotalSize() int64       { return r.TotalSize }
func (r *ResponseInfo) GetDownloadSpeed() float64 { return r.DownloadSpeed }

func New(url, fileName string, connections int, retryCount int) *DownloadRequest {
	return &DownloadRequest{
		Url:            url,
		FileName:       fileName,
		threads:        connections,
		RetryCount:     retryCount,
		StreamDownload: false,
	}
}

func (d *DownloadRequest) GetDownloadInfo() (int64, string, error) {
	resp, err := http.Head(d.Url)
	if err != nil {
		return -1, "", err
	}
	defer resp.Body.Close()
	contentDisposition := resp.Header.Get("Content-Disposition")
	fileNameHeader := strings.Split(contentDisposition, "filename=")
	var fileName = ""
	if len(fileName) > 1 {
		fileName = strings.Split(fileNameHeader[1], ";")[0]
	}
	return resp.ContentLength, fileName, nil
}

func (d *DownloadRequest) GetEachChunkSize(downloadSize int64) int64 {
	return downloadSize / int64(d.threads)
}

func (d *DownloadRequest) DownloadChunk(offset, endByte int64, resume bool,
	respSizeChan chan int64, resDown *ResumeDownload, workerId int) error {

	retryCount := 0

	var file *os.File
	var err error
	if resume {
		file, err = os.OpenFile(d.FileName, os.O_RDWR, 0644)
	} else {
		file, err = os.Create(d.FileName)
		resDown.DownloadChunks[workerId] = DownloadChunk{Start: offset, Offset: offset}
	}
	if err != nil {
		return err
	}

	defer file.Close()

	req, err := http.NewRequest("GET", d.Url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", offset, endByte))
	req.Header.Set("User-Agent", "Firefox")
	for key, value := range d.CustomHeaders {
		req.Header.Set(key, value)
	}

	client := GetHttpClient(time.Second * 60)

RETRY_LABEL:

	resp, err := client.Do(req)
	if err != nil {
		if retryCount < d.RetryCount {
			retryCount--
			goto RETRY_LABEL
		}
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		if retryCount < d.RetryCount {
			retryCount--
			time.Sleep(time.Second)
			goto RETRY_LABEL
		}
		return fmt.Errorf("server returned non-OK status: %v", resp.Status)
	}
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}
	var buffer = make([]byte, 32*1024)
	mustBreak := false
	for {
		size, err := resp.Body.Read(buffer)
		if err != nil {
			if err != io.EOF {
				if retryCount < d.RetryCount {
					retryCount--
					time.Sleep(time.Second)
					goto RETRY_LABEL
				}
				log.Printf("[-] Can't read response body: %v", err)
				break
			}
			if size <= 0 {
				mustBreak = true
				break
			}

		}

		respSizeChan <- int64(size)

		_, err = file.WriteAt(buffer[:size], offset)
		if err != nil {
			log.Printf("[-] Can't write response body to file: %v", err)
		}

		resDown.DownloadChunks[workerId].Offset = offset
		offset += int64(size)

		if mustBreak {
			break
		}
	}

	return nil
}

func (d *DownloadRequest) DownloadStream() error {

	retryCount := 0

	file, err := os.Create(d.FileName)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequest("GET", d.Url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Firefox")
	for key, value := range d.CustomHeaders {
		req.Header.Set(key, value)
	}
	client := GetHttpClient(time.Second * 60)

RETRY_LABEL:

	resp, err := client.Do(req)
	if err != nil {
		if retryCount < d.RetryCount {
			retryCount--
			goto RETRY_LABEL
		}
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		if retryCount < d.RetryCount {
			retryCount--
			time.Sleep(time.Second)
			goto RETRY_LABEL
		}
		return fmt.Errorf("server returned non-OK status: %v", resp.Status)
	}
	var buffer = make([]byte, 1024*1024)
	mustBreak := false
	for {
		size, err := resp.Body.Read(buffer)
		if err != nil {
			if err != io.EOF {
				if retryCount < d.RetryCount {
					retryCount--
					time.Sleep(time.Second)
					goto RETRY_LABEL
				}
				log.Printf("[-] Can't read response body: %v", err)
				break
			}
			if size <= 0 {
				mustBreak = true
				break
			}

		}
		_, err = file.Write(buffer[:size])
		if err != nil {
			log.Printf("[-] Can't write response body to file: %v", err)
		}
		if mustBreak {
			break
		}
	}

	return nil
}

func (resInfo *ResponseInfo) CalculateResponseInfo(downloadSize int64, recvBytesChan chan int64) {
	resInfo.DownloadSpeed = 0
	resInfo.TotalSize = downloadSize
	downloadPerSecond := int64(0)
	t := time.NewTicker(time.Second * 1)
	defer t.Stop()
	for {
		select {
		case size := <-recvBytesChan:
			resInfo.DownloadedSize += size
			downloadPerSecond += size
		case <-t.C:
			resInfo.DownloadSpeed = float64(downloadPerSecond)
			downloadPerSecond = 0
		}
	}
}

func (d *DownloadRequest) Start(downloadSize, chunkSize int64, responseInfo *ResponseInfo,
	wg *sync.WaitGroup, resDown *ResumeDownload, resume bool) error {
	if !resume {
		resDown.DownloadChunks = make([]DownloadChunk, d.threads)
	}

	var respSizeChan = make(chan int64)
	for i := 0; i < d.threads; i++ {
		offset := int64(i) * chunkSize
		endByte := offset + chunkSize - 1

		if i == d.threads-1 {
			endByte = downloadSize - 1
		}
		go func(offset, endByte int64, wg *sync.WaitGroup, workerId int) {
			if resume && endByte == (resDown.DownloadChunks[workerId].Offset) {
				return
			}
			if resume {
				offset = resDown.DownloadChunks[workerId].Offset
			}
			wg.Add(1)
			err := d.DownloadChunk(offset, endByte, resume, respSizeChan, resDown, workerId)
			wg.Done()
			if err != nil {
				log.Printf("[-] Error downloading chunk: %s", err.Error())
			}
		}(offset, endByte, wg, i)
	}
	go func(respChan chan int64) {
		responseInfo.CalculateResponseInfo(downloadSize, respChan)
	}(respSizeChan)
	return nil
}

func (d *DownloadRequest) StartStreamDownload() {
	d.DownloadStream()
}
