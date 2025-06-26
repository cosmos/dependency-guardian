package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile   string
	logLevel  string
	logFormat string
)

var rootCmd = &cobra.Command{
	Use:   "dependency-guardian",
	Short: "A tool to analyze dependency impacts in Pull Requests",
	Long: `Dependency Guardian analyzes GitHub Pull Requests to identify
dependency tree changes and their potential impact on high-level modules.
It helps developers understand the full scope of their changes and ensures
proper testing of affected components.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set up logger
		var level zapcore.Level
		if err := level.Set(logLevel); err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}

		var cfg zap.Config
		if logFormat == "json" {
			cfg = zap.NewProductionConfig()
		} else {
			cfg = zap.NewDevelopmentConfig()
			// More human-readable time format for text logs
			cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
		}

		cfg.Level = zap.NewAtomicLevelAt(level)
		logger, err := cfg.Build()
		if err != nil {
			return fmt.Errorf("failed to build logger: %w", err)
		}

		zap.ReplaceGlobals(logger)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .dependency-guardian.yml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Log format (text, json)")
} 