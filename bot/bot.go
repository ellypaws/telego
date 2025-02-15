package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"telegram-discord/bot/discord"
	"telegram-discord/bot/telegram"
)

type Bot struct {
	Discord  *discord.Bot
	Telegram *telegram.Bot
	Bots     []Bots
}

type Bots interface {
	Registrar
	Starter
	Stopper
}

type Registrar interface {
	Commands() error
	Handlers()
}

type Starter interface {
	Start() error
}

type Stopper interface {
	Stop() error
}

type Config struct {
	DiscordToken     string
	DiscordChannelID string

	TelegramToken     string
	TelegramChannelID string
	TelegramThreadID  string
}

func New(config Config) (*Bot, error) {
	discordBot, err := discord.New(config.DiscordToken, config.DiscordChannelID)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord bot: %w", err)
	}

	tgChannelID, err := strconv.ParseInt(config.TelegramChannelID, 10, 64)
	if err != nil {
		tgChannelID = 0
	}

	tgThreadID, err := strconv.ParseInt(config.TelegramThreadID, 10, 64)
	if err != nil {
		tgThreadID = 0
	}

	tgBot, err := telegram.New(config.TelegramToken, tgChannelID, int(tgThreadID))
	if err != nil {
		return nil, fmt.Errorf("error creating Telegram bot: %w", err)
	}

	return &Bot{
		Discord:  discordBot,
		Telegram: tgBot,
		Bots: []Bots{
			discordBot,
			tgBot,
		},
	}, nil
}

func (b *Bot) Start() error {
	var wg sync.WaitGroup
	for _, bot := range b.Bots {
		wg.Add(1)
		go func(bot Bots) {
			defer wg.Done()
			err := bot.Start()
			if err != nil {
				log.Printf("error starting %T: %v", bot, err)
				return
			}
			log.Printf("%T is running", bot)

			log.Printf("Registering commands for %T...", bot)
			err = bot.Commands()
			if err != nil {
				log.Printf("error registering %T commands: %v", bot, err)
				return
			}
			log.Printf("Commands registered for %T", bot)

			go bot.Handlers()
		}(bot)
	}
	wg.Wait()

	b.registerMainHandler()
	log.Println("Discord to Telegram mirroring bot is running. Press CTRL+C to exit.")
	return nil
}

func (b *Bot) Wait() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}

func (b *Bot) Shutdown() error {
	log.Println("Shutting down...")
	var wg sync.WaitGroup
	for _, registrar := range b.Bots {
		wg.Add(1)
		go func(registrar Bots) {
			defer wg.Done()
			err := registrar.Stop()
			if err != nil {
				log.Printf("error stopping %T: %v", registrar, err)
			}
		}(registrar)
	}
	wg.Wait()
	return nil
}
