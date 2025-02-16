package telegram

import (
	"fmt"
	"time"

	"gopkg.in/telebot.v4"
	"telegram-discord/lib"
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

func (b *Bot) Send(content any) error {
	if b.Channel == 0 {
		return fmt.Errorf("channel not set")
	}
	chat := &telebot.Chat{ID: b.Channel}
	_, err := b.Bot.Send(chat, content, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdownV2,
		ThreadID:  b.ThreadID,
	})
	if err != nil {
		b.logger.Error(
			"Failed to send message to Telegram",
			"error", err,
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
			"content", content,
		)
		return fmt.Errorf("error sending to telegram: %w", err)
	}

	b.logger.Info(
		"Message sent to Telegram",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
		"content", content,
	)
	return nil
}

func (b *Bot) handleSendToThisChannel(c telebot.Context) error {
	b.Channel = c.Chat().ID
	b.ThreadID = c.Message().ThreadID
	b.logger.Info(
		"Telegram channel registered",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
		"channel_title", c.Chat().Title,
		"user", c.Sender().Username,
	)

	if err := c.Delete(); err != nil {
		b.logger.Error(
			"Failed to delete command message",
			"error", err,
			"channel_id", b.Channel,
			"message_id", c.Message().ID,
		)
	}
	message, err := c.Bot().Send(
		c.Recipient(),
		"✅ Successfully registered this channel for message forwarding",
		&telebot.Topic{
			ThreadID: b.ThreadID,
		})
	if err != nil {
		b.logger.Error(
			"Failed to send confirmation message",
			"error", err,
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
		)
		return fmt.Errorf("error sending message: %w", err)
	}

	if err := lib.SetWithLog(b.logger, map[string]string{
		"TELEGRAM_CHANNEL_ID": fmt.Sprintf("%d", b.Channel),
		"TELEGRAM_THREAD_ID":  fmt.Sprintf("%d", b.ThreadID),
	}); err != nil {
		b.logger.Error(
			"Error setting environment variables",
			"error", err,
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
		)
		return fmt.Errorf("error setting environment variables: %w", err)
	}

	b.logger.Info(
		"Telegram channel registered",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
		"channel_title", c.Chat().Title,
		"user", c.Sender().Username,
	)

	time.AfterFunc(5*time.Second, func() {
		if err := b.Bot.Delete(message); err != nil {
			b.logger.Error(
				"Failed to delete confirmation message",
				"error", err,
				"channel_id", b.Channel,
				"message_id", message.ID,
			)
		}
	})
	return nil
}

func (b *Bot) handleUnsubscribe(c telebot.Context) error {
	if c.Chat().ID != b.Channel {
		return c.Send("This channel is not currently registered for message forwarding")
	}

	oldChannel := b.Channel
	oldThread := b.ThreadID
	b.Channel = 0
	b.logger.Info(
		"Telegram channel unregistered",
		"channel_id", oldChannel,
		"channel_title", c.Chat().Title,
		"thread_id", oldThread,
		"user", c.Sender().Username,
	)

	if err := lib.SetWithLog(b.logger, map[string]string{
		"TELEGRAM_CHANNEL_ID": "",
		"TELEGRAM_THREAD_ID":  "",
	}); err != nil {
		b.logger.Error("Error resetting environment variables", "error", err)
	}

	return c.Send("✅ Successfully unregistered this channel from message forwarding")
}
