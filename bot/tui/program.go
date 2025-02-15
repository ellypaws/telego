package tui

import (
	"telegram-discord/bot/tui/components/logger"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	telegramLogger = "telegram"
	discordLogger  = "discord"
)

type Model struct {
	loggers *logger.Stack
	width   int
	height  int
}

func NewModel() Model {
	return Model{
		loggers: logger.NewStack(telegramLogger, discordLogger),
	}
}

func (m Model) Init() tea.Cmd {
	return m.loggers.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.loggers.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	// Update loggers
	loggerModel, loggerCmd := m.loggers.Update(msg)
	if loggerStack, ok := loggerModel.(*logger.Stack); ok {
		m.loggers = loggerStack
	}
	cmd = loggerCmd

	return m, cmd
}

func (m Model) View() string {
	return m.loggers.View()
}

func Start() error {
	p := tea.NewProgram(
		NewModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
