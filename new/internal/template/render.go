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

// outputNameMap maps embedded template names to their rendered output names.
var outputNameMap = map[string]string{
	"env.tmpl":                ".env",
	"env.gateway.tmpl":        ".env.gateway",
	"nginx.conf.tmpl":         "nginx.conf",
	"settings.yml.tmpl":       "settings.yml",
	"docker-compose.yml.tmpl": "docker-compose.yml",
	"coredns-custom.yaml.tmpl": "coredns-custom.yaml",
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

			outPath := filepath.Join(outDir, outName)
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
		return os.WriteFile(filepath.Join(outDir, name), data, 0644)
	})
}
