package bot

import (
	"telegram-discord/lib"
	"telegram-discord/lib/parser"
	"telegram-discord/lib/parser/parserv5"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(b.mainHandler)
	b.Discord.Session.AddHandler(b.deleteMessageHandler)
	b.Discord.Session.AddHandler(b.messageUpdateHandler)
}

func (b *Bot) mainHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		b.Discord.Logger().Debug(
			"Skipping message - self message",
			"message_id", m.ID,
			"channel", lib.ChannelNameID(s, m.ChannelID),
			"author", lib.GetUsername(m),
		)
		return
	}

	if b.Discord.Channel == "" {
		b.Discord.Logger().Warn(
			"Skipping message - Discord channel not registered",
			"message_id", m.ID,
			"channel", lib.ChannelNameID(s, m.ChannelID),
			"author", lib.GetUsername(m),
		)
		return
	}

	if b.Telegram.Channel == 0 {
		b.Telegram.Logger().Warn(
			"Skipping message - Telegram channel not registered",
			"message_id", m.ID,
			"channel", lib.ChannelNameID(s, m.ChannelID),
			"author", lib.GetUsername(m),
		)
		return
	}

	if m.ChannelID != b.Discord.Channel {
		b.Discord.Logger().Debug(
			"Skipping message - wrong channel",
			"received_channel", lib.ChannelNameID(s, m.ChannelID),
			"target_channel", lib.ChannelNameID(s, b.Discord.Channel),
			"author", lib.GetUsername(m),
		)
		return
	}

	var (
		message *discordgo.Message = m.Message
	)

	if m.MessageReference != nil {
		b.Discord.Logger().Debug(
			"Processing message with reference",
			"message_id", m.ID,
			"reference_id", m.MessageReference.MessageID,
			"author", lib.GetUsername(m.MessageReference),
		)

		retrieve, err := b.Discord.Session.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
		if err != nil {
			b.Discord.Logger().Error(
				"Failed to retrieve referenced message",
				"error", err,
				"channel", lib.ChannelNameID(s, m.MessageReference.ChannelID),
				"message_id", m.MessageReference.MessageID,
				"author", lib.GetUsername(m.MessageReference),
			)
			return
		}
		message = retrieve
	}

	b.Discord.Logger().Debug(
		"Processing message",
		"message_id", message.ID,
		"channel", lib.ChannelNameID(s, message.ChannelID),
		"author", lib.GetUsername(message),
	)
	toSend := parser.Sendable(s, message, parserv5.Parse)
	if toSend == nil {
		b.Discord.Logger().Warn(
			"Skipping message - no content to forward",
			"message_id", message.ID,
			"channel", lib.ChannelNameID(s, message.ChannelID),
			"author", lib.GetUsername(message),
		)
		return
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
	} else {
		b.Discord.Set(message, reference)
		b.Discord.Logger().Info(
			"Successfully forwarded message to Telegram",
			"message_id", message.ID,
			"channel", lib.ChannelNameID(s, message.ChannelID),
			"author", lib.GetUsername(message),
			"content_length", len(message.Content),
		)
	}
}

func (b *Bot) deleteMessageHandler(s *discordgo.Session, m *discordgo.MessageDelete) {
	reference, ok := b.Discord.Get(m.Message)
	if !ok {
		b.Discord.Logger().Debug(
			"Message was deleted but not tracked",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m),
		)
		return
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
	} else {
		b.Discord.Unset(m.Message)
		b.Discord.Logger().Info(
			"Successfully deleted message from Telegram",
			"message_id", reference.Discord.ID,
			"channel", lib.ChannelNameID(s, reference.Discord.ChannelID),
			"author", lib.GetUsername(reference.Discord),
		)
	}
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	reference, ok := b.Discord.Get(m.Message)
	if !ok {
		b.Discord.Logger().Debug(
			"Message was updated but not tracked",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m.Message),
		)
		return
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
	toSend := parser.Sendable(s, m.Message, parserv5.Parse)
	if toSend == nil {
		b.Discord.Logger().Warn(
			"Skipping message - no content to edit",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m),
		)
		return
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
	} else {
		b.Discord.Set(m.Message, edited)
		b.Discord.Logger().Info(
			"Successfully edited message in Telegram",
			"message_id", m.Message.ID,
			"channel", lib.ChannelNameID(s, m.Message.ChannelID),
			"author", lib.GetUsername(m),
		)
	}
}
