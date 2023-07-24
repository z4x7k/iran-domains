package main

import (
	"github.com/go-telegram/bot/models"
)

func shouldDiscard(update *models.Update) bool {
	checks := []func() bool{
		func() bool { return update.Message == nil },
		func() bool { return update.Message.From.IsBot },
		func() bool { return update.Message.Chat.IsForum },
		func() bool { return update.Message.Chat.Type != "private" },
	}
	for _, fn := range checks {
		if fn() {
			return true
		}
	}

	return false
}
