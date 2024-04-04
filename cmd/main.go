package main

import (
	"log/slog"
	"os"

	"go.openfort.xyz/shield/pkg/logger"

	"go.openfort.xyz/shield/cmd/cli"
)

func main() {
	slog.Info("Starting OpenFort Shield")
	rootCmd := cli.NewCmdRoot()
	if err := rootCmd.Execute(); err != nil {
		slog.Info("Error executing command", logger.Error(err))
		os.Exit(1)
	}
}
