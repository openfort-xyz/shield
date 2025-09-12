package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"go.openfort.xyz/shield/pkg/logger"

	"go.openfort.xyz/shield/cmd/cli"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found or error loading it", logger.Error(err))
	}
	
	slog.Info("Starting OpenFort Shield")
	rootCmd := cli.NewCmdRoot()
	if err := rootCmd.Execute(); err != nil {
		slog.Info("Error executing command", logger.Error(err))
		os.Exit(1)
	}
}
