package main

import (
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func (h *Handler) loggerFromUpdate(update *models.Update) zerolog.Logger {
	return h.logger.
		With().
		Int64("chat_id", update.Message.Chat.ID).
		Str("chat_username", update.Message.Chat.Username).
		Str("user_username", update.Message.From.Username).
		Str("user_first_name", update.Message.From.FirstName).
		Str("user_last_name", update.Message.From.LastName).
		Int64("user_id", update.Message.From.ID).
		Int("message_date", update.Message.Date).
		Str("message_text", update.Message.Text).
		Logger()
}
