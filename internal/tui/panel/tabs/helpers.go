package tabs

// maskValue masks a secret value for display. Shows "****" if non-empty, "(not set)" if empty.
func maskValue(s string) string {
	if s == "" {
		return "(not set)"
	}
	return "****"
}

// displayValue shows the value or "(not set)" if empty.
func displayValue(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
