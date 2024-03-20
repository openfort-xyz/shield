package main

import (
	"go.openfort.xyz/shield/cmd/cli"
	"log/slog"
	"os"
)

func main() {
	slog.Info("Starting OpenFort Shield")
	rootCmd := cli.NewCmdRoot()
	if err := rootCmd.Execute(); err != nil {
		slog.Info("Error executing command", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
