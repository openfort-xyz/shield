package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/openfort-xyz/shield/pkg/logger"

	"github.com/openfort-xyz/shield/cmd/cli"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found or error loading it", logger.Error(err))
	}

	// After godotenv, so LOG_LEVEL can come from .env as well as the environment.
	// Loggers built before this point pick the new level up too.
	if err := logger.ConfigureFromEnv(); err != nil {
		slog.Warn("Ignoring invalid log level, defaulting to info", logger.Error(err))
	}

	slog.Info("Starting OpenFort Shield")
	rootCmd := cli.NewCmdRoot()
	if err := rootCmd.Execute(); err != nil {
		slog.Info("Error executing command", logger.Error(err))
		os.Exit(1)
	}
}
