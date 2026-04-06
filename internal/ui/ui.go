package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type ProgressReporter interface {
	Start(title string, total int)
	Update(current int)
	Inc()
	Finish(message string)
}

type NoopProgressReporter struct{}

func (n *NoopProgressReporter) Start(title string, total int) {}
func (n *NoopProgressReporter) Update(current int)           {}
func (n *NoopProgressReporter) Inc()                         {}
func (n *NoopProgressReporter) Finish(message string)        {}

type TeaProgressReporter struct {
	title   string
	total   int
	current int
	prog    *tea.Program
}

type progressMsg float64

func NewTeaProgressReporter() *TeaProgressReporter {
	return &TeaProgressReporter{}
}

func (t *TeaProgressReporter) Start(title string, total int) {
	t.title = title
	t.total = total
	t.current = 0

	m := model{
		title:    title,
		progress: progress.New(progress.WithDefaultGradient()),
		total:    total,
	}

	t.prog = tea.NewProgram(m)
	go func() {
		if _, err := t.prog.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running progress: %v\n", err)
		}
	}()
}

func (t *TeaProgressReporter) Update(current int) {
	t.current = current
	if t.prog != nil {
		t.prog.Send(progressMsg(float64(t.current) / float64(t.total)))
	}
}

func (t *TeaProgressReporter) Inc() {
	t.current++
	t.Update(t.current)
}

func (t *TeaProgressReporter) Finish(message string) {
	if t.prog != nil {
		t.prog.Quit()
		if message != "" {
			fmt.Println(SuccessStyle.Render(message))
		}
	}
}

type model struct {
	title    string
	progress progress.Model
	total    int
	current  int
	err      error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case progressMsg:
		var cmds []tea.Cmd
		if msg >= 1.0 {
			cmds = append(cmds, tea.Quit)
		}
		cmds = append(cmds, m.progress.SetPercent(float64(msg)))
		return m, tea.Batch(cmds...)
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error() + "\n"
	}
	pad := "  "
	return "\n" +
		pad + m.title + "\n" +
		pad + m.progress.View() + "\n\n"
}
