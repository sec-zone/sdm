package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/sec-zone/sdm/downloader"
	"github.com/sec-zone/sdm/tui"
	"github.com/sec-zone/sdm/utils"
)

var wg = &sync.WaitGroup{}
var interruptChan = make(chan bool)

var (
	flUrl              = flag.String("u", "", "Specify the target URL for downloading file.")
	flFileName         = flag.String("o", "", "Specify the file name of downloaded file. If not specified the program try to get filename from content-disposition header.")
	flConnectionsCount = flag.Int("c", 10, "Number of connections to download")
	flRetry            = flag.Int("r", 10, "Specify the number of times to retry downloading if error is encountered.")
	flHeader           = flag.String("H", "", "Specify the custom header you want sent when downloading. Example: x-authorization-header=abcd;x-test-header=foo")
)

func init() {
	flag.Parse()
}

func main() {
	var isResuming = false

	downReq := downloader.New(*flUrl, *flFileName, *flConnectionsCount, *flRetry)
	if *flHeader != "" {
		downReq.CustomHeaders = utils.ParseHeaders(*flHeader)
	}
	downloadSize, serverFileName, err := downReq.GetDownloadInfo()
	fmt.Printf("  Total Download size: %.2f kB\n", float64(downloadSize)/1024.0)
	if err != nil {
		log.Printf("[-] Can't get download size. Err: %s\n", err.Error())
		downReq.StreamDownload = true
	}
	if downReq.FileName == "" && serverFileName == "" {
		log.Println("[-] The server didn't specified the filename.")
		downUrlSegments := strings.Split(downReq.Url, "/")
		tmp := strings.Split(downUrlSegments[len(downUrlSegments)-1], "?")
		if len(tmp) > 1 {
			downReq.FileName = tmp[0]
		} else {
			downReq.FileName = downUrlSegments[len(downUrlSegments)-1]
		}
	} else if downReq.FileName == "" {
		downReq.FileName = serverFileName
	}
	if downReq.StreamDownload {
		log.Printf("[+] Download has been started.\n\n Because I haven't download size, it's not possible to create progress bar.")
		downReq.StartStreamDownload()
		log.Printf("[+] Done.")
		return
	}
	chunkSize := downReq.GetEachChunkSize(downloadSize)
	fmt.Printf("  Each Chunk size: %2.fKB\n", float64(chunkSize)/1024)

	var responseInfo = &downloader.ResponseInfo{}
	var resumeDownload = &downloader.ResumeDownload{}

	cacheFile, err := os.ReadFile(fmt.Sprintf(".%s.cache", downReq.FileName))
	if err == nil {
		json.Unmarshal(cacheFile, resumeDownload)
		isResuming = true
		fmt.Printf("  Download resumed...\n")
	}

	// if download is resumed then we should calculated downloaded size
	if isResuming {
		totalDownloaded := int64(0) // bytes
		for i := 0; i < len(resumeDownload.DownloadChunks); i++ {
			totalDownloaded += resumeDownload.DownloadChunks[i].Offset - resumeDownload.DownloadChunks[i].Start
		}
		responseInfo.DownloadedSize = totalDownloaded
	}
	go func() {
		err = downReq.Start(downloadSize, chunkSize, responseInfo, wg, resumeDownload, isResuming)
		if err != nil {
			log.Fatalf("[-] Can't start download. Err: %s\n", err.Error())
		}
	}()

	go func() {
		for {
			select {
			case <-interruptChan:
				fmt.Printf("\n  Total size: %d\tDownload Size: %d\n", responseInfo.TotalSize, responseInfo.DownloadedSize)
				if responseInfo.DownloadedSize >= responseInfo.TotalSize {
					fmt.Printf("\nDownloaded Successfully.\n")
				} else {
					fmt.Printf("\nDownload Paused. You can resume download by repeating it's command.\n")
					resDownJson, _ := json.Marshal(resumeDownload)
					os.WriteFile(fmt.Sprintf(".%s.cache", downReq.FileName), resDownJson, 0644)
				}
			}
		}
	}()
	tui.Start(downloadSize, responseInfo, interruptChan)
	wg.Wait()
	if responseInfo.DownloadedSize >= responseInfo.TotalSize {
		os.Remove(fmt.Sprintf(".%s.cache", downReq.FileName))
	}
}
