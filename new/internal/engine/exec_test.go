package engine

import (
	"context"
	"testing"
	"time"
)

func TestRunWithTimeout_Success(t *testing.T) {
	err := RunWithTimeout(context.Background(), "echo-test", 5*time.Second, "echo", "hello")
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}
}

func TestRunWithTimeout_Timeout(t *testing.T) {
	err := RunWithTimeout(context.Background(), "sleep-test", 100*time.Millisecond, "sleep", "10")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestRunWithTimeout_BadCommand(t *testing.T) {
	err := RunWithTimeout(context.Background(), "bad-cmd", 5*time.Second, "nonexistent-command-xyz")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}
