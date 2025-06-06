package telegram

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	Bot      *telebot.Bot
	Channel  int64
	ThreadID int

	logger *log.Logger
}

func New(token string, channel int64, threadID int, output io.Writer) (*Bot, error) {
	settings := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(settings)
	if err != nil {
		return nil, fmt.Errorf("error creating telegram bot: %w", err)
	}

	logger := log.NewWithOptions(output,
		log.Options{
			Level:           log.DebugLevel,
			ReportTimestamp: true,
			Prefix:          "[Telegram]",
		},
	)
	logger.SetColorProfile(termenv.TrueColor)

	return &Bot{
		Bot:      bot,
		Channel:  channel,
		ThreadID: threadID,

		logger: logger,
	}, nil
}

func (b *Bot) Logger() *log.Logger {
	return b.logger
}

func (b *Bot) Start() error {
	go b.Bot.Start()
	b.logger.Info(
		"Telegram bot started",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
	)
	return nil
}

func (b *Bot) Stop() error {
	b.logger.Info("Stopping Telegram bot")
	b.Bot.Stop()
	return nil
}
