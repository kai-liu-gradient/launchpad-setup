package wizard

import "testing"

func TestExpressWizard_InitialState(t *testing.T) {
	m := New(Express)
	if m.mode != Express {
		t.Errorf("mode = %v, want Express", m.mode)
	}
	if m.done {
		t.Error("wizard should not be done initially")
	}
}

func TestCustomWizard_InitialState(t *testing.T) {
	m := New(Custom)
	if m.mode != Custom {
		t.Errorf("mode = %v, want Custom", m.mode)
	}
}

func TestWizard_ViewRendersWithoutPanic(t *testing.T) {
	m := New(Express)
	m.Init()
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}
