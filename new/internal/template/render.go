package template

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

// RenderContext is passed to all templates as the top-level data.
type RenderContext struct {
	Config  *config.Config
	Secrets *secrets.Secrets
	Derived *DerivedValues
	Runtime *RuntimeValues
}

// outputNameMap maps embedded template names to their rendered output paths
// (relative to outDir). Paths with slashes create subdirectories automatically.
var outputNameMap = map[string]string{
	"env.tmpl":                 "launchpad/.env",
	"env.gateway.tmpl":         "gateway/.env",
	"nginx.conf.tmpl":          "nginx/nginx.conf",
	"settings.yml.tmpl":        "launchpad/config/settings.yml",
	"docker-compose.yml.tmpl":  "docker-compose.yml",
	"coredns-custom.yaml.tmpl": "coredns-custom.yaml",
	"crontab":                  "launchpad/cron/crontab",
	// Kyverno files are static (no .tmpl) — copied as-is by RenderAll.
	// They contain Kyverno's own {{ }} syntax which would conflict with Go templates.
}

// RenderAll walks the embedded template files, renders .tmpl files with ctx,
// and copies static files as-is to outDir.
func RenderAll(ctx *RenderContext, outDir string) error {
	return fs.WalkDir(Templates, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		name := filepath.Base(path)

		if strings.HasSuffix(name, ".tmpl") {
			// Template file — parse and execute
			data, readErr := fs.ReadFile(Templates, path)
			if readErr != nil {
				return readErr
			}

			tmpl, parseErr := template.New(name).Parse(string(data))
			if parseErr != nil {
				return parseErr
			}

			outName := outputNameMap[name]
			if outName == "" {
				outName = strings.TrimSuffix(name, ".tmpl")
			}

			// Skip settings.yml if it already exists (preserve user edits)
			if outName == "launchpad/config/settings.yml" {
				outPath := filepath.Join(outDir, outName)
				if _, statErr := os.Stat(outPath); statErr == nil {
					return nil
				}
			}

			outPath := filepath.Join(outDir, outName)
			if err := prepareOutputPath(outPath); err != nil {
				return err
			}
			f, createErr := os.Create(outPath)
			if createErr != nil {
				return createErr
			}
			defer f.Close()

			return tmpl.Execute(f, ctx)
		}

		// Static file — copy as-is
		data, readErr := fs.ReadFile(Templates, path)
		if readErr != nil {
			return readErr
		}
		outName := outputNameMap[name]
		if outName == "" {
			outName = name
		}
		outPath := filepath.Join(outDir, outName)
		if err := prepareOutputPath(outPath); err != nil {
			return err
		}
		return os.WriteFile(outPath, data, 0644)
	})
}

// prepareOutputPath ensures parent directories exist and removes any
// conflicting directory at outPath (Docker may create directories for
// missing volume mount targets).
func prepareOutputPath(outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	info, err := os.Stat(outPath)
	if err == nil && info.IsDir() {
		if err := os.RemoveAll(outPath); err != nil {
			return err
		}
	}
	return nil
}
