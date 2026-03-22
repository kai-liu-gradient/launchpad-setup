// new/cmd/launchpad/main.go
package main

import (
	"fmt"
	"os"

	"github.com/gradient8/launchpad/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
