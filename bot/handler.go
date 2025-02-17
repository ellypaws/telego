package bot

import (
	"telegram-discord/bot/parser"
	"telegram-discord/bot/parser/parserv5"

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
			"channel_id", m.ChannelID,
			"author", m.Author.Username,
		)
		return
	}

	if b.Discord.Channel == "" {
		b.Discord.Logger().Warn(
			"Skipping message - Discord channel not registered",
			"message_id", m.ID,
			"channel_id", m.ChannelID,
			"author", m.Author.Username,
		)
		return
	}

	if b.Telegram.Channel == 0 {
		b.Telegram.Logger().Warn(
			"Skipping message - Telegram channel not registered",
			"message_id", m.ID,
			"channel_id", m.ChannelID,
			"author", m.Author.Username,
		)
		return
	}

	if m.ChannelID != b.Discord.Channel {
		b.Discord.Logger().Debug(
			"Skipping message - wrong channel",
			"received_channel", m.ChannelID,
			"target_channel", b.Discord.Channel,
			"author", m.Author.Username,
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
			"author", m.Author.Username,
		)

		retrieve, err := b.Discord.Session.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
		if err != nil {
			b.Discord.Logger().Error(
				"Failed to retrieve referenced message",
				"error", err,
				"channel_id", m.MessageReference.ChannelID,
				"message_id", m.MessageReference.MessageID,
				"author", m.Author.Username,
			)
			return
		}
		message = retrieve
	}

	b.Discord.Logger().Debug(
		"Processing message",
		"message_id", message.ID,
		"channel_id", message.ChannelID,
		"author", message.Author.Username,
	)
	toSend := parser.Sendable(s, message, parserv5.Parse)
	if toSend == nil {
		b.Telegram.Logger().Debug(
			"Skipping message - no content to forward",
			"message_id", message.ID,
			"channel_id", message.ChannelID,
			"author", message.Author.Username,
		)
		return
	}

	b.Discord.Logger().Info(
		"Forwarding message to Telegram",
		"message_id", message.ID,
		"channel_id", message.ChannelID,
		"author", message.Author.Username,
		"content_length", len(message.Content),
	)

	reference, err := b.Telegram.Send(toSend)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to forward message to Telegram",
			"error", err,
			"message_id", message.ID,
			"channel_id", message.ChannelID,
			"author", message.Author.Username,
			"content_length", len(message.Content),
		)
	} else {
		b.Discord.Set(message, reference)
		b.Discord.Logger().Info(
			"Successfully forwarded message to Telegram",
			"message_id", message.ID,
			"channel_id", message.ChannelID,
			"author", message.Author.Username,
			"content_length", len(message.Content),
		)
	}
}

func (b *Bot) deleteMessageHandler(s *discordgo.Session, m *discordgo.MessageDelete) {
	reference, ok := b.Discord.Get(m.Message)
	if !ok {
		b.Discord.Logger().Debug(
			"Skipping message - not tracked",
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
		return
	}

	b.Discord.Logger().Debug(
		"Tracked message was deleted",
		"message_id", reference.Telegram.ID,
		"chat_id", reference.Telegram.Chat.ID,
	)
	b.Telegram.Logger().Debug(
		"Tracked message was deleted",
		"message_id", reference.Discord.ID,
		"channel_id", reference.Discord.ChannelID,
	)

	err := b.Telegram.Bot.Delete(reference.Telegram)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to delete message from Telegram",
			"error", err,
			"message_id", reference.Discord.ID,
			"channel_id", reference.Discord.ChannelID,
		)
		b.Telegram.Logger().Error(
			"Failed to delete message from Telegram",
			"error", err,
			"message_id", reference.Telegram.ID,
			"chat_id", reference.Telegram.Chat.ID,
		)
	} else {
		b.Discord.Unset(m.Message)
		b.Discord.Logger().Info(
			"Successfully deleted message from Telegram",
			"message_id", reference.Discord.ID,
			"channel_id", reference.Discord.ChannelID,
		)
		b.Telegram.Logger().Info(
			"Successfully deleted message from Telegram",
			"message_id", reference.Telegram.ID,
			"chat_id", reference.Telegram.Chat.ID,
		)
	}
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	reference, ok := b.Discord.Get(m.Message)
	if !ok {
		b.Discord.Logger().Debug(
			"Skipping message - not tracked",
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
		return
	}

	b.Discord.Logger().Debug(
		"Tracked message was updated",
		"message_id", reference.Discord.ID,
		"channel_id", reference.Discord.ChannelID,
	)
	b.Telegram.Logger().Debug(
		"Tracked message was updated",
		"message_id", reference.Telegram.ID,
		"chat_id", reference.Telegram.Chat.ID,
	)

	b.Discord.Logger().Debug(
		"Processing message",
		"message_id", m.ID,
		"channel_id", m.ChannelID,
		"author", m.Author.Username,
	)
	toSend := parser.Sendable(s, m.Message, parserv5.Parse)
	if toSend == nil {
		b.Telegram.Logger().Debug(
			"Skipping message - no content to forward",
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
		return
	}
	edited, err := b.Telegram.Bot.Edit(reference.Telegram, toSend)
	if err != nil {
		b.Discord.Logger().Error(
			"Failed to delete message from Telegram",
			"error", err,
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
		b.Telegram.Logger().Error(
			"Failed to delete message from Telegram",
			"error", err,
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
	} else {
		b.Discord.Set(m.Message, edited)
		b.Discord.Logger().Info(
			"Successfully edited message in Telegram",
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
		b.Telegram.Logger().Info(
			"Successfully edited message in Telegram",
			"message_id", m.Message.ID,
			"channel_id", m.Message.ChannelID,
			"author", m.Message.Author.Username,
		)
	}
}
