package telegram

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"log"
	"time"
)

const (
	cmdSendToThisChannel = "/setchannel"
	cmdUnsubscribe       = "/unsetchannel"
)

func (b *Bot) Commands() error {
	return nil
}

func (b *Bot) Handlers() {
	b.Bot.Handle(cmdSendToThisChannel, b.handleSendToThisChannel)
	b.Bot.Handle(cmdUnsubscribe, b.handleUnsubscribe)
}

func (b *Bot) Send(text string) error {
	if b.Channel == 0 {
		return fmt.Errorf("channel not set")
	}
	chat := &telebot.Chat{ID: b.Channel}
	_, err := b.Bot.Send(chat, text, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdownV2,
	})
	if err != nil {
		return fmt.Errorf("error sending to telegram: %w", err)
	}

	return nil
}

func (b *Bot) handleSendToThisChannel(c telebot.Context) error {
	b.Channel = c.Chat().ID
	b.ThreadID = c.Message().ThreadID
	log.Printf("Registered new Telegram channel: %s (%d : %d)", c.Chat().Title, b.Channel, b.ThreadID)

	if err := c.Send("✅ Successfully registered this channel for message forwarding"); err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	c.DeleteAfter(5 * time.Second)
	return nil
}

func (b *Bot) handleUnsubscribe(c telebot.Context) error {
	if c.Chat().ID != b.Channel {
		return c.Send("This channel is not currently registered for message forwarding")
	}

	b.Channel = 0 // Reset channel ID
	log.Printf("Unregistered Telegram channel: %d", c.Chat().ID)

	return c.Send("✅ Successfully unregistered this channel from message forwarding")
}
