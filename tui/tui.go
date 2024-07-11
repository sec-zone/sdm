package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

const (
	padding  = 2
	maxWidth = 100
)

type progressMsg float64

type progressErrMsg struct{ err error }

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

type Model struct {
	PW            *ProgressWriter
	Progress      progress.Model
	Err           error
	InterruptChan chan bool
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.InterruptChan <- true
			time.Sleep(time.Second)
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.Progress.Width = msg.Width - padding*2 - 4
		if m.Progress.Width > maxWidth {
			m.Progress.Width = maxWidth
		}
		return m, nil

	case progressErrMsg:
		m.Err = msg.err
		return m, tea.Quit

	case progressMsg:
		var cmds []tea.Cmd

		if msg >= 1.0 {
			cmds = append(cmds, tea.Sequence(finalPause(), tea.Quit))
		}

		cmds = append(cmds, m.Progress.SetPercent(float64(msg)))
		return m, tea.Batch(cmds...)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m Model) View() string {
	if m.Err != nil {
		return "Error downloading: " + m.Err.Error() + "\n"
	}

	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.Progress.View() + "\n\n" +
		pad + fmt.Sprintf("Speed: %.2f MB/S\n", m.PW.downSpeed/1024.0/1024.0) +
		pad + fmt.Sprintf("Downloaded: %.2f KB / %.2f KB\n", float64(m.PW.downloaded)/1024.0, float64(m.PW.total)/1024.0) +
		pad + fmt.Sprintf("ETA: %.2f M\n\n", float64(m.PW.total-m.PW.downloaded)/m.PW.downSpeed/60) +
		pad + helpStyle("Press ctr+c to pause.\n\n")
}
