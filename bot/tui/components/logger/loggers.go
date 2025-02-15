package logger

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Stack struct {
	loggers map[string]*Logger
	width   int
	height  int
	padding int
}

func NewStack(loggers ...string) *Stack {
	if len(loggers) == 0 {
		loggers = []string{"log"}
	}

	stack := &Stack{
		loggers: make(map[string]*Logger),
	}

	for _, logger := range loggers {
		stack.loggers[logger] = NewLogger()
	}

	return stack
}

func (s *Stack) GetLoggers() map[string]*Logger {
	return s.loggers
}

func (s *Stack) GetLogger(name string) *Logger {
	return s.loggers[name]
}

func (s *Stack) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, l := range s.loggers {
		cmds = append(cmds, l.Init())
	}
	return tea.Batch(cmds...)
}

func (s *Stack) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Only propagate if dimensions changed
		if s.width != msg.Width || s.height != msg.Height {
			s.width = msg.Width
			s.height = msg.Height

			width := s.calculateWidth()
			for i, l := range s.loggers {
				if model, cmd := l.Update(tea.WindowSizeMsg{
					Width:  width,
					Height: msg.Height - s.padding,
				}); cmd != nil {
					s.loggers[i] = model.(*Logger)
					cmds = append(cmds, cmd)
				}
			}
		}

	default:
		for i, l := range s.loggers {
			if model, cmd := l.Update(msg); cmd != nil {
				s.loggers[i] = model.(*Logger)
				cmds = append(cmds, cmd)
			}
		}
	}

	return s, tea.Batch(cmds...)
}

func (s *Stack) View() string {
	views := make([]string, len(s.loggers))
	for _, l := range s.loggers {
		views = append(views, l.View())
		views = append(views, strings.Repeat(" ", s.padding))
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, views[:len(views)-1]...)
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

func (s *Stack) SetSize(width, height int) {
	s.width = width
	s.height = height

	// Calculate height for each logger
	loggerHeight := (height - (s.padding * (len(s.loggers) - 1))) / len(s.loggers)

	// Set size for each logger
	currentY := 0
	for _, l := range s.loggers {
		l.SetSize(width, loggerHeight)
		currentY += loggerHeight + s.padding
	}
}
