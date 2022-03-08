package main

import (
	"os"

	"github.com/TylerHendrickson/mydyndns/cmd/mydyndns/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
