package progress

import (
	"testing"

	"github.com/gradient8/launchpad/internal/engine"
)

func TestNew_InitialState(t *testing.T) {
	m := New([]string{"Step A", "Step B", "Step C"})
	if len(m.steps) != 3 {
		t.Errorf("steps count = %d, want 3", len(m.steps))
	}
	for _, s := range m.steps {
		if s.Status != engine.Pending {
			t.Errorf("initial step status = %v, want Pending", s.Status)
		}
	}
	if m.done {
		t.Error("model should not be done initially")
	}
	if m.err != nil {
		t.Error("model should have no error initially")
	}
}

func TestUpdate_StepRunning(t *testing.T) {
	m := New([]string{"Step A", "Step B"})
	updated, _ := m.Update(StepEventMsg{Step: "Step A", Status: engine.Running})
	model := updated.(Model)
	if model.steps[0].Status != engine.Running {
		t.Errorf("Step A status = %v, want Running", model.steps[0].Status)
	}
	if model.current != 0 {
		t.Errorf("current = %d, want 0", model.current)
	}
}

func TestUpdate_StepDone(t *testing.T) {
	m := New([]string{"Step A", "Step B"})
	updated, _ := m.Update(StepEventMsg{Step: "Step A", Status: engine.Done})
	model := updated.(Model)
	if model.steps[0].Status != engine.Done {
		t.Errorf("Step A status = %v, want Done", model.steps[0].Status)
	}
	if model.done {
		t.Error("model should not be done with one step remaining")
	}
}

func TestUpdate_AllDone(t *testing.T) {
	m := New([]string{"Step A", "Step B"})
	m1, _ := m.Update(StepEventMsg{Step: "Step A", Status: engine.Done})
	m2, _ := m1.Update(StepEventMsg{Step: "Step B", Status: engine.Done})
	model := m2.(Model)
	if !model.done {
		t.Error("model should be done when all steps complete")
	}
}

func TestUpdate_StepFailed(t *testing.T) {
	m := New([]string{"Step A"})
	updated, _ := m.Update(StepEventMsg{
		Step:   "Step A",
		Status: engine.Failed,
		Err:    errTest,
	})
	model := updated.(Model)
	if model.steps[0].Status != engine.Failed {
		t.Errorf("Step A status = %v, want Failed", model.steps[0].Status)
	}
	if model.err == nil {
		t.Error("model should have error on failure")
	}
}

func TestView_NotEmpty(t *testing.T) {
	m := New([]string{"Step A"})
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
	if !contains(view, "Step A") {
		t.Error("View should contain step name")
	}
	if !contains(view, "Deploying") {
		t.Error("View should contain title")
	}
}

func TestView_ShowsCompletion(t *testing.T) {
	m := New([]string{"Step A"})
	updated, _ := m.Update(StepEventMsg{Step: "Step A", Status: engine.Done})
	model := updated.(Model)
	view := model.View()
	if !contains(view, "complete") {
		t.Error("View should show completion message")
	}
}

func TestView_ShowsError(t *testing.T) {
	m := New([]string{"Step A"})
	updated, _ := m.Update(StepEventMsg{
		Step:   "Step A",
		Status: engine.Failed,
		Err:    errTest,
	})
	model := updated.(Model)
	view := model.View()
	if !contains(view, "failed") {
		t.Error("View should show failure message")
	}
}

func TestStepView_Render(t *testing.T) {
	tests := []struct {
		name   string
		status engine.StepStatus
		want   string
	}{
		{"pending", engine.Pending, "Test Step"},
		{"running", engine.Running, "Test Step"},
		{"done", engine.Done, "Test Step"},
		{"failed", engine.Failed, "Test Step"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := StepView{Name: "Test Step", Status: tt.status}
			rendered := sv.Render()
			if !contains(rendered, tt.want) {
				t.Errorf("Render() = %q, should contain %q", rendered, tt.want)
			}
		})
	}
}

// helpers

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
