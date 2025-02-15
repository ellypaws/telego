package main

import (
	"log"
	"os"

	"telegram-discord/bot"

	"github.com/joho/godotenv"
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
		TelegramThreadID:  os.Getenv("TELEGRAM_THREAD_ID"),
	})
	if err != nil {
		log.Fatalf("error creating bot: %v", err)
	}

	go func() {
		err = b.Start()
		if err != nil {
			log.Fatalf("error starting bot: %v", err)
		}
	}()

	b.Wait()
	if err := b.Shutdown(); err != nil {
		log.Fatalf("error shutting down bot: %v", err)
	}
}
