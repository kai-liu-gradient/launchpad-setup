package tabs

import (
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type TelegramTab struct {
	BotToken    string
	BotUsername string
	ChatID      string
}

func NewTelegramTab(cfg *config.Config) *TelegramTab {
	return &TelegramTab{
		BotToken:    cfg.Telegram.BotToken,
		BotUsername: cfg.Telegram.BotUsername,
		ChatID:      cfg.Telegram.ChatID,
	}
}

func (t *TelegramTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Bot Token").
				Description("Obtain from @BotFather on Telegram").
				EchoMode(huh.EchoModePassword).
				Value(&t.BotToken),
			huh.NewInput().
				Title("Bot Username").
				Description("e.g. MyLaunchpadBot (without @)").
				Value(&t.BotUsername),
			huh.NewInput().
				Title("Chat ID").
				Description("Group or channel chat ID for notifications").
				Value(&t.ChatID),
		),
	)
}

func (t *TelegramTab) Apply(cfg *config.Config) {
	cfg.Telegram.BotToken = t.BotToken
	cfg.Telegram.BotUsername = t.BotUsername
	cfg.Telegram.ChatID = t.ChatID
}
