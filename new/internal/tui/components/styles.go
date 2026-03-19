package components

import "github.com/charmbracelet/lipgloss"

var (
	// Brand colors
	PrimaryColor  = lipgloss.Color("#6C5CE7")
	SuccessColor  = lipgloss.Color("#00B894")
	WarningColor  = lipgloss.Color("#FDCB6E")
	ErrorColor    = lipgloss.Color("#E74C3C")
	MutedColor    = lipgloss.Color("#555555")
	SubtitleColor = lipgloss.Color("#888888")

	// Text styles
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8C291"))
	SubtitleStyle = lipgloss.NewStyle().Foreground(SubtitleColor)
	SuccessStyle  = lipgloss.NewStyle().Foreground(SuccessColor)
	ErrorStyle    = lipgloss.NewStyle().Foreground(ErrorColor)
	MutedStyle    = lipgloss.NewStyle().Foreground(MutedColor)

	// Component styles
	StepDone    = SuccessStyle.Render("✓")
	StepRunning = lipgloss.NewStyle().Foreground(WarningColor).Render("⠋")
	StepPending = MutedStyle.Render("○")
	StepFailed  = ErrorStyle.Render("✗")
)
