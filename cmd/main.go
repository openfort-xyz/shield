package main

import (
	"log/slog"
	"os"

	"go.openfort.xyz/shield/cmd/cli"
)

func main() {
	slog.Info("Starting OpenFort Shield")
	rootCmd := cli.NewCmdRoot()
	if err := rootCmd.Execute(); err != nil {
		slog.Info("Error executing command", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
