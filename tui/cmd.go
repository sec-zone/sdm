package tui

import (
	"fmt"
	"time"

	"os"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

var p *tea.Program

type ProgressWriter struct {
	total      int64
	downloaded int64
	onProgress func(float64)
	downSpeed  float64
}

type IResInfo interface {
	GetTotalSize() int64
	GetDownloadedSize() int64
	GetDownloadSpeed() float64
}

func (pw *ProgressWriter) Start(resInfo IResInfo) {
	// TeeReader calls pw.Write() each time a new response is received
	pw.downloaded = 0
	for {
		pw.downloaded = resInfo.GetDownloadedSize()
		pw.downSpeed = resInfo.GetDownloadSpeed()
		pw.Write()
		time.Sleep(time.Second)
	}
}

func (pw *ProgressWriter) Write() error {
	if pw.total > 0 && pw.onProgress != nil {
		pw.onProgress(float64(pw.downloaded) / float64(pw.total))
	}
	return nil
}

func Start(downloadSize int64, resInfo IResInfo, interruptChan chan bool) {
	pw := &ProgressWriter{
		total:      downloadSize,
		downloaded: 0,
		onProgress: func(ratio float64) {
			p.Send(progressMsg(ratio))
		},
	}
	m := Model{
		PW:            pw,
		Progress:      progress.New(progress.WithDefaultGradient()),
		InterruptChan: interruptChan,
	}
	// Start Bubble Tea
	p = tea.NewProgram(m)

	go pw.Start(resInfo)

	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}

}
