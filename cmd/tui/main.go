package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"log"
	"os"
	"telegram-discord/bot"
	"telegram-discord/lib"
	"telegram-discord/tui"
	"telegram-discord/tui/components/logger"
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
		DiscordToken:     os.Getenv("DISCORD_TOKEN"),
		DiscordChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
		DiscordLogger:    writers[0],

		TelegramToken:     os.Getenv("TELEGRAM_TOKEN"),
		TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		TelegramThreadID:  os.Getenv("TELEGRAM_THREAD_ID"),
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
