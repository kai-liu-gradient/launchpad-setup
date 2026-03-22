// Package importfiles embeds template git repos (.tar.gz) and YAML template
// definitions (.yaml) that are imported into Gitea during deployment.
// This is separate from internal/template which embeds deployment config
// templates (.tmpl files) used for rendering docker-compose, nginx, etc.
package importfiles

import "embed"

//go:embed all:files
var Files embed.FS
