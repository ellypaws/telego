package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"telegram-discord/bot"
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func main() {
	b, err := bot.New(bot.Config{
		DiscordToken:      os.Getenv("DISCORD_TOKEN"),
		DiscordChannelID:  os.Getenv("DISCORD_CHANNEL_ID"),
		TelegramToken:     os.Getenv("TELEGRAM_TOKEN"),
		TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
	})
	if err != nil {
		log.Fatalf("error creating bot: %v", err)
	}

	err = b.Start()
	if err != nil {
		log.Fatalf("error starting bot: %v", err)
	}
}
