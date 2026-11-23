// Command hyrts is a hybrid regression test selection tool for Go modules.
package main

import (
	"os"

	"github.com/dangdtr/go-hyrts/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
