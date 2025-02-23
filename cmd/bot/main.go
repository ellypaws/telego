package main

import (
	"log"
	"os"

	"telegram-discord/bot"
	"telegram-discord/lib"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func main() {
	defer lib.LogOutput(os.Stdout)()

	b, err := bot.New(bot.Config{
		DiscordToken:      os.Getenv(lib.EnvDiscordToken),
		DiscordChannelID:  os.Getenv(lib.EnvDiscordChannel),
		TelegramToken:     os.Getenv(lib.EnvTelegramToken),
		TelegramChannelID: os.Getenv(lib.EnvTelegramChannel),
		TelegramThreadID:  os.Getenv(lib.EnvTelegramThread),
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
