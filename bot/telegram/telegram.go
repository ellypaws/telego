package telegram

import (
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
