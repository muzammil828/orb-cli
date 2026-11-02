package main

import (
	"log/slog"
	"os"

	"orb/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
