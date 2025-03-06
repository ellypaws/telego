package bot

import (
	"telegram-discord/lib"
	"telegram-discord/lib/parser/parserv5"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(Chain(
		b.mainHandler,
		SkipperMiddleware(b, OnlyBots),
		RetryMiddleware[*discordgo.MessageCreate](b, 3),
		// WhitelistMiddleware(whitelist),
	))

	b.Discord.Session.AddHandler(Chain(
		b.deleteMessageHandler,
		RetryMiddleware[*discordgo.MessageDelete](b, 3),
	))

	b.Discord.Session.AddHandler(Chain(
		b.messageUpdateHandler,
		RetryMiddleware[*discordgo.MessageUpdate](b, 3),
	))
}

func (b *Bot) mainHandler(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID {
		b.Discord.Logger().Debug(
			"Skipping message - self message",
			"message_id", m.ID,
			"channel", lib.ChannelNameID(s, m.ChannelID),
			"author", lib.GetUsername(m),
		)
		return nil
	}
	if b.Discord.Channel == "" {
		b.Discord.Logger().Warn(
			"Skipping message - Discord channel not registered",
			"message_id", m.ID,
			"channel", lib.ChannelNameID(s, m.ChannelID),
			"author", lib.GetUsername(m),
		)
		return nil
	}
	if b.Telegram.Channel == 0 {
		b.Telegram.Logger().Warn(
			"Skipping message - Telegram channel not registered",
			"message_id", m.ID,
			"channel", lib.ChannelNameID(s, m.ChannelID),
			"author", lib.GetUsername(m),
		)
		return nil
	}

	if m.ChannelID != b.Discord.Channel {
		b.Discord.Logger().Debug(
			"Skipping message - wrong channel",
			"received_channel", lib.ChannelNameID(s, m.ChannelID),
			"target_channel", lib.ChannelNameID(s, b.Discord.Channel),
			"author", lib.GetUsername(m),
		)
		return nil
	}

	message := m.Message
	if m.MessageReference != nil {
		retrieve, err := lib.GetReference(b.Discord.Logger(), s, m)
		if err != nil {
			return err
		}
		message = retrieve
	}

	b.Discord.Logger().Debug(
		"Processing message",
		"message_id", message.ID,
		"channel", lib.ChannelNameID(s, message.ChannelID),
		"author", lib.GetUsername(message),
	)
	toSend, err := parserv5.Sendable(s, message, parserv5.Parse)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to process message",
			"error", err,
			"message_id", message.ID,
			"channel", lib.ChannelNameID(s, message.ChannelID),
			"author", lib.GetUsername(message),
		)
		return err
	}
	if toSend == nil {
		b.Discord.Logger().Warn(
			"Skipping message - no content to forward",
			"message_id", message.ID,
			"channel", lib.ChannelNameID(s, message.ChannelID),
			"author", lib.GetUsername(message),
		)
		return nil
	}

	b.Discord.Logger().Info(
		"Forwarding message to Telegram",
		"message_id", message.ID,
		"channel", lib.ChannelNameID(s, message.ChannelID),
		"author", lib.GetUsername(message),
		"content_length", len(message.Content),
	)
	reference, err := b.Telegram.Send(toSend)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to forward message to Telegram",
			"error", err,
			"message_id", message.ID,
			"channel", lib.ChannelNameID(s, message.ChannelID),
			"author", lib.GetUsername(message),
			"content_length", len(message.Content),
		)
		return err
	}
	b.Discord.Set(message, reference)
	b.Discord.Logger().Info(
		"Successfully forwarded message to Telegram",
		"message_id", message.ID,
		"channel", lib.ChannelNameID(s, message.ChannelID),
		"author", lib.GetUsername(message),
		"content_length", len(message.Content),
	)
	return nil
}

func (b *Bot) deleteMessageHandler(s *discordgo.Session, m *discordgo.MessageDelete) error {
	reference, ok := b.Discord.Get(m.Message)
	if !ok {
		b.Discord.Logger().Debug(
			"Message was deleted but not tracked",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m.Message),
		)
		return nil
	}
	b.Discord.Logger().Debug(
		"Message was deleted, deleting from Telegram",
		"message_id", m.Message.ID,
		"channel", lib.ChannelNameID(s, m.Message.ChannelID),
		"author", lib.GetUsername(m.Message),
	)
	err := b.Telegram.Delete(reference.Telegram)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to delete message from Telegram",
			"error", err,
			"message_id", reference.Discord.ID,
			"channel", lib.ChannelNameID(s, reference.Discord.ChannelID),
			"author", lib.GetUsername(reference.Discord),
		)
		return err
	}
	b.Discord.Unset(m.Message)
	b.Discord.Logger().Info(
		"Successfully deleted message from Telegram",
		"message_id", reference.Discord.ID,
		"channel", lib.ChannelNameID(s, reference.Discord.ChannelID),
		"author", lib.GetUsername(reference.Discord),
	)
	return nil
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) error {
	reference, ok := b.Discord.Get(m.Message)
	if !ok {
		b.Discord.Logger().Debug(
			"Message was updated but not tracked",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m.Message),
		)
		return nil
	}
	b.Discord.Logger().Debug(
		"Message was updated, updating in Telegram",
		"message_id", reference.Discord.ID,
		"channel", lib.ChannelNameID(s, reference.Discord.ChannelID),
		"author", lib.GetUsername(reference.Discord),
	)
	b.Discord.Logger().Debug(
		"Processing message",
		"message_id", m.ID,
		"channel", lib.ChannelNameID(s, m.ChannelID),
		"author", lib.GetUsername(m),
	)
	toSend, err := parserv5.Sendable(s, m.Message, parserv5.Parse)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to process message",
			"error", err,
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m),
		)
		return err
	}
	if toSend == nil {
		b.Discord.Logger().Warn(
			"Skipping message - no content to edit",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m),
		)
		return nil
	}
	edited, err := b.Telegram.Edit(reference.Telegram, toSend)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to edit message in Telegram",
			"error", err,
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m),
		)
		return err
	}
	b.Discord.Set(m.Message, edited)
	b.Discord.Logger().Info(
		"Successfully edited message in Telegram",
		"message_id", m.Message.ID,
		"channel", lib.ChannelNameID(s, m.Message.ChannelID),
		"author", lib.GetUsername(m),
	)
	return nil
}
