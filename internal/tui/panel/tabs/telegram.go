package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type TelegramTab struct {
	BotToken    string
	BotUsername string
}

func NewTelegramTab(cfg *config.Config) *TelegramTab {
	return &TelegramTab{
		BotToken:    cfg.Telegram.BotToken,
		BotUsername: cfg.Telegram.BotUsername,
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
		),
	)
}

func (t *TelegramTab) Groups() []*huh.Group {
	return []*huh.Group{
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
		).Title("Telegram"),
	}
}

func (t *TelegramTab) View() string {
	return fmt.Sprintf(
		"  Bot Token:    %s\n  Bot Username: %s",
		maskValue(t.BotToken), displayValue(t.BotUsername))
}

func (t *TelegramTab) Apply(cfg *config.Config) {
	cfg.Telegram.BotToken = t.BotToken
	cfg.Telegram.BotUsername = t.BotUsername
}
