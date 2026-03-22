package engine

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

const hostsFile = "/etc/hosts"
const hostsMarker = "# launchpad-managed"

func (e *Engine) configureEtcHosts(_ context.Context) error {
	launchpadDomain := e.cfg.Subdomain + "." + e.cfg.Domain
	giteaDomain := e.cfg.Subdomain + "-gitea." + e.cfg.Domain

	entries := map[string]string{
		launchpadDomain: "127.0.0.1",
		giteaDomain:     "127.0.0.1",
	}

	return addHostEntries(entries)
}

func addHostEntries(entries map[string]string) error {
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", hostsFile, err)
	}

	existing := string(data)
	var toAdd []string
	for domain, ip := range entries {
		if !strings.Contains(existing, domain) {
			toAdd = append(toAdd, fmt.Sprintf("%s  %s %s", ip, domain, hostsMarker))
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening %s for append: %w", hostsFile, err)
	}
	defer f.Close()

	for _, line := range toAdd {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}

// RemoveHostEntries removes all launchpad-managed entries from /etc/hosts.
func RemoveHostEntries() error {
	f, err := os.Open(hostsFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, hostsMarker) {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(hostsFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
