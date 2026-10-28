package main

import (
	"fmt"
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
	flushLogs := lib.LogOutput(os.Stdout)
	err := run()
	if err != nil {
		log.Printf("exiting: %v", err)
	}
	flushLogs()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	b, err := bot.New(bot.Config{
		DiscordToken:      os.Getenv(lib.EnvDiscordToken),
		DiscordChannelID:  os.Getenv(lib.EnvDiscordChannel),
		TelegramToken:     os.Getenv(lib.EnvTelegramToken),
		TelegramChannelID: os.Getenv(lib.EnvTelegramChannel),
		TelegramThreadID:  os.Getenv(lib.EnvTelegramThread),
	})
	if err != nil {
		return fmt.Errorf("error creating bot: %w", err)
	}

	go func() {
		if err := b.Start(); err != nil {
			log.Fatalf("error starting bot: %v", err)
		}
	}()

	waitErr := b.Wait()
	if err := b.Shutdown(); err != nil {
		return fmt.Errorf("error shutting down bot: %w", err)
	}
	return waitErr
}
