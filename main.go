package main

import (
	"os"

	"github.com/bishop-bot/ib-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
