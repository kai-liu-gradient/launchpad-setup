package tabs

import "fmt"

func boolDisplay(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func displayValue(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

func listDisplay(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("%v", items)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
