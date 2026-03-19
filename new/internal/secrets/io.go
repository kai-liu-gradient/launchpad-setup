package secrets

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Save marshals secrets to YAML and writes to path with 0600 permissions.
func Save(s *Secrets, path string) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling secrets: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing secrets file: %w", err)
	}
	return nil
}

// Load reads a YAML secrets file from path and returns the parsed Secrets.
func Load(path string) (*Secrets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading secrets file: %w", err)
	}
	var s Secrets
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing secrets file: %w", err)
	}
	return &s, nil
}
