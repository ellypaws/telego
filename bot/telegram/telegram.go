package telegram

import (
	"fmt"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type Bot struct {
	Bot     *tgbotapi.BotAPI
	Channel int64
}

func New(token string, channel int64) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Bot:     bot,
		Channel: channel,
	}, nil
}

func (b *Bot) Send(text string) error {
	msg := tgbotapi.NewMessage(b.Channel, text)
	_, err := b.Bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error sending to telegram: %w", err)
	}

	return nil
}
