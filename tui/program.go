package tui

import (
	"telegram-discord/tui/components/logger"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Stopper interface {
	Shutdown() error
}

type Model struct {
	loggers  tea.Model
	shutdown Stopper
	width    int
	height   int
}

func NewModel(loggers *logger.Stack, shutdown Stopper) Model {
	return Model{
		loggers:  loggers,
		shutdown: shutdown,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loggers.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.width == msg.Width && m.height == msg.Height {
			return m, nil
		}
		m.width = msg.Width
		m.height = msg.Height

		return m.propagate(msg, cmd)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, m.Shutdown
		}
	case finished:
		return m, tea.Quit
	}

	return m.propagate(msg, cmd)
}

type finished struct{}

func (m Model) Shutdown() tea.Msg {
	if err := m.shutdown.Shutdown(); err != nil {
		m.propagate(logger.Message{Message: "error shutting down bot: " + err.Error()}, nil)
	}
	time.Sleep(5 * time.Second)
	return finished{}
}

func (m Model) propagate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, m.loggers.(*logger.Stack).Len())
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.loggers, cmd = m.loggers.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.loggers.View())
}
