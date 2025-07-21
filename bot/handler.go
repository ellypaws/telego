package bot

import (
	"errors"
	"fmt"

	"telegram-discord/lib"
	"telegram-discord/lib/parser/parserv5"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/telebot.v4"
)

func (b *Bot) registerMainHandler() {
	b.Discord.Session.AddHandler(Chain(
		b.mainHandler,
		SkipperMiddleware(b.Discord.Logger(), OnlyBots),
		RetryMiddleware[*discordgo.MessageCreate](b.Discord.Logger(), 3, telebot.ErrEmptyText, telebot.ErrEmptyMessage),
		// WhitelistMiddleware(whitelist),
	))

	b.Discord.Session.AddHandler(Chain(
		b.deleteMessageHandler,
		RetryMiddleware[*discordgo.MessageDelete](b.Discord.Logger(), 3),
	))

	b.Discord.Session.AddHandler(Chain(
		b.messageUpdateHandler,
		RetryMiddleware[*discordgo.MessageUpdate](b.Discord.Logger(), 3, telebot.ErrMessageNotModified, telebot.ErrSameMessageContent),
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
	var toReply *telebot.Message
	if m.MessageReference != nil {
		switch m.MessageReference.Type {
		case discordgo.MessageReferenceTypeDefault:
			reference, ok := b.Discord.Get(m.MessageReference.MessageID)
			if ok {
				toReply = reference.Telegram
			} else {
				b.Discord.Logger().Warn("Could not find message reference for reply",
					"message_id", message.ID,
					"reference_id", m.MessageReference.MessageID,
				)
			}
		case discordgo.MessageReferenceTypeForward:
			retrieve, err := lib.GetReference(b.Discord.Logger(), s, m)
			if err != nil {
				return err
			}
			message = retrieve
		}
	}

	b.Discord.Logger().Debug(
		"Processing message",
		"message_id", message.ID,
		"channel", lib.ChannelNameID(s, message.ChannelID),
		"author", lib.GetUsername(message),
	)
	toSend, err := parserv5.Sendable(s, message, parserv5.Parser(s, message))
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

	options := &telebot.SendOptions{
		ReplyTo:   toReply,
		ParseMode: telebot.ModeMarkdownV2,
		ThreadID:  b.Telegram.ThreadID,
	}
	if m.Poll != nil {
		options.ReplyMarkup = &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{{
					Text: "VOTE HERE (Discord)",
					URL:  fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID),
				}},
			},
		}
	}
	reference, err := b.Telegram.Send(toSend, options)
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
	reference, ok := b.Discord.Get(m.Message.ID)
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
	b.Discord.Unset(m.Message.ID)
	b.Discord.Logger().Info(
		"Successfully deleted message from Telegram",
		"message_id", reference.Discord.ID,
		"channel", lib.ChannelNameID(s, reference.Discord.ChannelID),
		"author", lib.GetUsername(reference.Discord),
	)
	return nil
}

func (b *Bot) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) error {
	reference, ok := b.Discord.Get(m.Message.ID)
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
	toSend, err := parserv5.Sendable(s, m.Message, parserv5.Parser(s, m.Message))
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
		if errors.Is(err, telebot.ErrSameMessageContent) || errors.Is(err, telebot.ErrMessageNotModified) {
			return nil
		}
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
