package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"telegram-discord/bot/tui/components/logger"
)

type Model struct {
	loggers tea.Model
	width   int
	height  int
}

func NewModel(loggers *logger.Stack) Model {
	return Model{
		loggers: loggers,
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
			return m, tea.Quit
		}
	}

	return m.propagate(msg, cmd)
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
