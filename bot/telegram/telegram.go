package telegram

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/log"
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

	return &Bot{
		Bot:      bot,
		Channel:  channel,
		ThreadID: threadID,

		logger: log.New(output),
	}, nil
}

func (b *Bot) Logger() *log.Logger {
	return b.logger
}

func (b *Bot) Start() error {
	go b.Bot.Start()
	return nil
}

func (b *Bot) Stop() error {
	b.Bot.Stop()
	return nil
}
