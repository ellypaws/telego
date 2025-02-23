package main

import (
	"log"
	"os"

	"telegram-discord/bot"
	"telegram-discord/lib"
	"telegram-discord/tui"
	"telegram-discord/tui/components/logger"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

const (
	telegramLogger = "telegram"
	discordLogger  = "discord"
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func main() {
	loggers := logger.NewStack(discordLogger, telegramLogger)

	writers, closer, err := lib.NewLogWriters(loggers.Get(discordLogger), loggers.Get(telegramLogger))
	if err != nil {
		log.Fatalf("error creating log writers: %v", err)
	}
	defer closer()

	b, err := bot.New(bot.Config{
		DiscordToken:     os.Getenv(lib.EnvDiscordToken),
		DiscordChannelID: os.Getenv(lib.EnvDiscordChannel),
		DiscordLogger:    writers[0],

		TelegramToken:     os.Getenv(lib.EnvTelegramToken),
		TelegramChannelID: os.Getenv(lib.EnvTelegramChannel),
		TelegramThreadID:  os.Getenv(lib.EnvTelegramThread),
		TelegramLogger:    writers[1],
	})
	if err != nil {
		log.Fatalf("error creating bot: %v", err)
	}

	p := tea.NewProgram(
		tui.NewModel(loggers, b),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	go b.Start()
	p.Run()

	if err := b.Shutdown(); err != nil {
		log.Fatalf("error shutting down bot: %v", err)
	}
}
