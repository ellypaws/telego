package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"log"
	"os"
	"telegram-discord/bot"
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

	b, err := bot.New(bot.Config{
		DiscordToken:     os.Getenv("DISCORD_TOKEN"),
		DiscordChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
		DiscordLogger:    loggers.Get(discordLogger),

		TelegramToken:     os.Getenv("TELEGRAM_TOKEN"),
		TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		TelegramThreadID:  os.Getenv("TELEGRAM_THREAD_ID"),
		TelegramLogger:    loggers.Get(telegramLogger),
	})
	if err != nil {
		log.Fatalf("error creating bot: %v", err)
	}

	p := tea.NewProgram(
		tui.NewModel(loggers),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	go p.Run()
	go b.Start()

	b.Wait()

	if err := b.Shutdown(); err != nil {
		log.Fatalf("error shutting down bot: %v", err)
	}
}
