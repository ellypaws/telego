package telegram

import (
	"fmt"
	"time"

	"telegram-discord/lib"

	"gopkg.in/telebot.v4"
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

func (b *Bot) Send(content any) (*telebot.Message, error) {
	if b.Channel == 0 {
		b.logger.Warn("Cannot send message - channel not set")
		return nil, fmt.Errorf("channel not set")
	}

	b.logger.Debug(
		"Sending message to Telegram",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
		"content_type", fmt.Sprintf("%T", content),
	)
	chat := &telebot.Chat{ID: b.Channel}
	reference, err := b.Bot.Send(chat, content, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdownV2,
		ThreadID:  b.ThreadID,
	})
	if err != nil {
		b.logger.Error(
			"Failed to send message",
			"error", err,
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
			"content_type", fmt.Sprintf("%T", content),
		)
		return nil, fmt.Errorf("error sending to telegram: %w", err)
	}

	b.logger.Info(
		"Message sent successfully",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
		"content_type", fmt.Sprintf("%T", content),
	)
	return reference, nil
}

func (b *Bot) handleSendToThisChannel(c telebot.Context) error {
	if b.Channel == c.Chat().ID && b.ThreadID == c.Message().ThreadID {
		b.logger.Warn(
			"Channel already registered for message forwarding",
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
			"channel_title", c.Chat().Title,
			"user", c.Sender().Username,
		)
		return b.tempReply(c, "This channel is already registered for message forwarding")
	}
	b.Channel = c.Chat().ID
	b.ThreadID = c.Message().ThreadID

	err := b.tempReply(c, "✅ Successfully registered this channel for message forwarding")
	if err != nil {
		return err
	}

	if err := lib.SetWithLog(b.logger, map[string]string{
		lib.EnvTelegramChannel: fmt.Sprintf("%d", b.Channel),
		lib.EnvTelegramThread:  fmt.Sprintf("%d", b.ThreadID),
	}); err != nil {
		b.logger.Error(
			"Failed to save channel configuration",
			"error", err,
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
			"user", c.Sender().Username,
		)
		return fmt.Errorf("error setting environment variables: %w", err)
	}

	b.logger.Info(
		"Channel registered for message forwarding",
		"channel_id", b.Channel,
		"thread_id", b.ThreadID,
		"channel_title", c.Chat().Title,
		"user", c.Sender().Username,
	)

	return nil
}

func (b *Bot) handleUnsubscribe(c telebot.Context) error {
	if c.Chat().ID != b.Channel {
		return b.tempReply(c, "This channel is not currently registered for message forwarding")
	}

	if err := lib.SetWithLog(b.logger, map[string]string{
		lib.EnvTelegramChannel: "",
		lib.EnvTelegramThread:  "",
	}); err != nil {
		b.logger.Error(
			"Failed to save channel configuration",
			"error", err,
			"old_channel_id", b.Channel,
			"old_thread_id", b.ThreadID,
			"user", c.Sender().Username,
		)
		return fmt.Errorf("error setting environment variables: %w", err)
	}

	err := b.tempReply(c, "✅ Successfully unregistered this channel from message forwarding")
	if err != nil {
		return err
	}

	b.logger.Info(
		"Channel unregistered from message forwarding",
		"old_channel_id", b.Channel,
		"old_thread_id", b.ThreadID,
		"channel_title", c.Chat().Title,
		"user", c.Sender().Username,
	)

	b.Channel = 0
	b.ThreadID = 0

	return nil
}

func (b *Bot) tempReply(c telebot.Context, content string) error {
	message, err := c.Bot().Send(
		c.Recipient(),
		content,
		&telebot.Topic{ThreadID: b.ThreadID},
		&telebot.SendOptions{ReplyTo: c.Message()},
	)

	if err != nil {
		b.logger.Error(
			"Failed to send confirmation message",
			"error", err,
			"channel_id", b.Channel,
			"thread_id", b.ThreadID,
			"user", c.Sender().Username,
		)
		return fmt.Errorf("error sending message: %w", err)
	}

	time.AfterFunc(5*time.Second, func() {
		if err := b.Bot.Delete(message); err != nil {
			b.logger.Warn(
				"Failed to delete confirmation message",
				"error", err,
				"channel_id", b.Channel,
				"message_id", message.ID,
				"user", message.Sender.Username,
			)
		}

		if err := c.Delete(); err != nil {
			b.logger.Warn(
				"Failed to delete command message",
				"error", err,
				"channel_id", b.Channel,
				"message_id", c.Message().ID,
				"user", c.Sender().Username,
			)
		}
	})

	return nil
}
