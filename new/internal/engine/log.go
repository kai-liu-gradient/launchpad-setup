package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DeployLogger writes all deployment output to a log file.
type DeployLogger struct {
	file *os.File
	mu   sync.Mutex
}

// NewDeployLogger creates a deploy log file in the output directory.
func NewDeployLogger(outputDir string) (*DeployLogger, error) {
	logPath := filepath.Join(outputDir, "deploy.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("creating deploy log: %w", err)
	}

	// Write header
	fmt.Fprintf(f, "=== AniLaunchpad Deploy Log ===\n")
	fmt.Fprintf(f, "Started: %s\n\n", time.Now().Format(time.RFC3339))

	return &DeployLogger{file: f}, nil
}

// Write implements io.Writer.
func (l *DeployLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Write(p)
}

// LogStep writes a step event to the log.
func (l *DeployLogger) LogStep(step string, status string, detail string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.file, "[%s] [%s] %s", timestamp, status, step)
	if detail != "" {
		fmt.Fprintf(l.file, " — %s", detail)
	}
	fmt.Fprintln(l.file)
}

// Close closes the log file.
func (l *DeployLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.file, "\n=== Deploy finished: %s ===\n", time.Now().Format(time.RFC3339))
	return l.file.Close()
}

// TeeWriter returns a writer that writes to both dst and the deploy log.
func (l *DeployLogger) TeeWriter(dst io.Writer) io.Writer {
	return io.MultiWriter(dst, l)
}
