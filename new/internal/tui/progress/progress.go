package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/tui/components"
)

type StepEventMsg engine.StepEvent
type TickMsg time.Time

type Model struct {
	steps     []StepView
	current   int
	spinner   spinner.Model
	progress  progress.Model
	startTime time.Time
	elapsed   time.Duration
	done      bool
	err       error
	eventCh   <-chan engine.StepEvent
}

func New(stepNames []string, eventCh <-chan engine.StepEvent) Model {
	steps := make([]StepView, len(stepNames))
	for i, name := range stepNames {
		steps[i] = StepView{Name: name, Status: engine.Pending}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(components.WarningColor)

	p := progress.New(progress.WithDefaultGradient())

	return Model{
		steps:     steps,
		spinner:   s,
		progress:  p,
		startTime: time.Now(),
		eventCh:   eventCh,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickCmd(), WaitForEvents(m.eventCh))
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" && (m.done || m.err != nil) {
			return m, tea.Quit
		}
	case TickMsg:
		if m.done || m.err != nil {
			return m, nil // stop ticking
		}
		m.elapsed = time.Since(m.startTime)
		return m, tickCmd()
	case StepEventMsg:
		for i := range m.steps {
			if m.steps[i].Name == msg.Step {
				m.steps[i].Status = msg.Status
				m.steps[i].Detail = msg.Detail
				if msg.Status == engine.Running {
					m.current = i
					m.steps[i].Duration = 0
				}
				if msg.Status == engine.Done {
					m.steps[i].Duration = m.elapsed // approximate
				}
				if msg.Status == engine.Failed {
					m.err = msg.Err
					m.steps[i].Detail = msg.Err.Error()
				}
				break
			}
		}
		// Check if all done
		allDone := true
		for _, s := range m.steps {
			if s.Status != engine.Done && s.Status != engine.Failed {
				allDone = false
				break
			}
		}
		m.done = allDone
		if m.done || m.err != nil {
			return m, tea.Quit
		}
		return m, WaitForEvents(m.eventCh)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(components.TitleStyle.Render("Deploying AniLaunchpad..."))
	b.WriteString("\n\n")

	// Step list
	for _, step := range m.steps {
		b.WriteString(step.Render())
		b.WriteString("\n")
	}

	// Progress bar
	b.WriteString("\n")
	total := len(m.steps)
	completed := 0
	for _, s := range m.steps {
		if s.Status == engine.Done {
			completed++
		}
	}
	pct := float64(completed) / float64(total)
	b.WriteString(m.progress.ViewAs(pct))
	b.WriteString("\n")
	b.WriteString(components.MutedStyle.Render(
		fmt.Sprintf("Step %d/%d · Elapsed: %s", completed, total, m.elapsed.Round(time.Second)),
	))
	b.WriteString("\n")

	if m.done {
		b.WriteString("\n")
		b.WriteString(components.SuccessStyle.Render("✓ Deployment complete!"))
		b.WriteString("\n")
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(components.ErrorStyle.Render("✗ Deployment failed. See error above."))
		b.WriteString(components.MutedStyle.Render("\nPress q to exit"))
		b.WriteString("\n")
	}

	return b.String()
}

// WaitForEvents returns a tea.Cmd that reads from the event channel.
func WaitForEvents(ch <-chan engine.StepEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return StepEventMsg(ev)
	}
}
