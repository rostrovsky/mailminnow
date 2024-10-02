package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/rostrovsky/mailminnow/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.StampMilli,
			NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
		}),
	))

	cobra.OnInitialize(initConfig)

	rootCmd := &cobra.Command{
		Use:   "mailminnow",
		Short: "A simple SMTP server with web UI",
		Run:   server.RunServer,
	}

	rootCmd.PersistentFlags().Int("smtp-port", 1025, "SMTP server port")
	rootCmd.PersistentFlags().Int("http-port", 8025, "HTTP server port")
	rootCmd.PersistentFlags().String("domain", "localhost", "Server domain")

	viper.BindPFlag("smtp_port", rootCmd.PersistentFlags().Lookup("smtp-port"))
	viper.BindPFlag("http_port", rootCmd.PersistentFlags().Lookup("http-port"))
	viper.BindPFlag("domain", rootCmd.PersistentFlags().Lookup("domain"))

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("SMTP_SERVER")
	viper.AutomaticEnv()
}
