package logger

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Stack struct {
	loggers []*Logger
	width   int
	height  int
	padding int
}

func NewStack(names ...string) *Stack {
	if len(names) == 0 {
		names = []string{"log"}
	}

	const padding = 3
	loggers := make([]*Logger, len(names))
	for i, logger := range names {
		loggers[i] = NewLogger(logger, padding)
	}

	return &Stack{
		loggers: loggers,
		padding: padding,
	}
}

func (s *Stack) Len() int {
	return len(s.loggers)
}

func (s *Stack) Get(name string) *Logger {
	for _, l := range s.loggers {
		if l.Title == name {
			return l
		}
	}
	return nil
}

func (s *Stack) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, l := range s.loggers {
		cmds = append(cmds, l.Init())
	}
	return tea.Batch(cmds...)
}

func (s *Stack) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if s.width == msg.Width && s.height == msg.Height {
			return s, nil
		}
		s.width = msg.Width
		s.height = msg.Height

		width := s.calculateWidth()
		return s.propagate(tea.WindowSizeMsg{
			Width:  width - s.padding,
			Height: msg.Height,
		}, cmd)
	default:
		return s.propagate(msg, cmd)
	}
}

func (s *Stack) propagate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, s.Len())
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	for i, l := range s.loggers {
		if model, cmd := l.Update(msg); cmd != nil {
			s.loggers[i] = model.(*Logger)
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

func (s *Stack) View() string {
	views := make([]string, len(s.loggers))
	for _, l := range s.loggers {
		views = append(views, l.View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, views...)
}

func calculateRatio(numLoggers int) float64 {
	if numLoggers == 0 {
		return 0
	}

	return 1.0 / float64(numLoggers)
}

// calculateWidth returns the pixel widths for each logger based on current window size
func (s *Stack) calculateWidth() int {
	numLoggers := len(s.loggers)
	ratio := calculateRatio(numLoggers)

	return int(float64(s.width) * ratio)
}
