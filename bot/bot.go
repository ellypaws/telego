package bot

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"telegram-discord/bot/discord"
	"telegram-discord/bot/telegram"

	"github.com/charmbracelet/log"
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
	Logger
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

type Logger interface {
	Logger() *log.Logger
}

type Config struct {
	DiscordToken     string
	DiscordChannelID string
	DiscordLogger    io.Writer

	TelegramToken     string
	TelegramChannelID string
	TelegramThreadID  string
	TelegramLogger    io.Writer
}

func New(config Config) (*Bot, error) {
	if config.DiscordToken == "" {
		return nil, fmt.Errorf("discord token is required")
	}
	if config.TelegramToken == "" {
		return nil, fmt.Errorf("telegram token is required")
	}
	discordBot, err := discord.New(config.DiscordToken, config.DiscordChannelID, config.DiscordLogger)
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

	tgBot, err := telegram.New(config.TelegramToken, tgChannelID, int(tgThreadID), config.TelegramLogger)
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
			bot.Logger().Debug(
				"Starting bot",
				"type", fmt.Sprintf("%T", bot),
			)
			err := bot.Start()
			if err != nil {
				bot.Logger().Error(
					"Failed to start bot",
					"type", fmt.Sprintf("%T", bot),
					"error", err,
				)
				return
			}
			bot.Logger().Info(
				"Bot started successfully",
				"type", fmt.Sprintf("%T", bot),
			)

			bot.Logger().Debug(
				"Registering commands",
				"type", fmt.Sprintf("%T", bot),
			)
			err = bot.Commands()
			if err != nil {
				bot.Logger().Error(
					"Failed to register commands",
					"type", fmt.Sprintf("%T", bot),
					"error", err,
				)
				return
			}
			bot.Logger().Debug(
				"Commands registered successfully",
				"type", fmt.Sprintf("%T", bot),
			)

			bot.Handlers()
		}(bot)
	}
	wg.Wait()

	b.registerMainHandler()
	b.Discord.Logger().Info("Message mirroring bot is running")
	return nil
}

func (b *Bot) Wait() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}

func (b *Bot) Shutdown() error {
	b.Discord.Logger().Info("Shutting down bots")
	var wg sync.WaitGroup
	for _, registrar := range b.Bots {
		wg.Add(1)
		go func(registrar Bots) {
			defer wg.Done()
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				err := registrar.Stop()
				if err != nil {
					registrar.Logger().Error(
						"Failed to stop bot",
						"type", fmt.Sprintf("%T", registrar),
						"error", err,
					)
				} else {
					registrar.Logger().Info(
						"Bot stopped successfully",
						"type", fmt.Sprintf("%T", registrar),
					)
				}
			}()
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				registrar.Logger().Error(
					"Bot did not stop in time",
					"type", fmt.Sprintf("%T", registrar),
				)
			}
		}(registrar)
	}
	wg.Wait()
	return nil
}
