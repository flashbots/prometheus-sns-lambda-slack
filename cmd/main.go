package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "development"
)

const (
	appName = "prometheus-sns-lambda-slack"
)

func main() {
	var logFormat, logLevel string

	flagLogLevel := &cli.StringFlag{
		Destination: &logLevel,
		EnvVars:     []string{"LOG_LEVEL"},
		Name:        "log-level",
		Usage:       "logging level",
		Value:       "info",
	}

	flagLogMode := &cli.StringFlag{
		Destination: &logFormat,
		EnvVars:     []string{"LOG_MODE"},
		Name:        "log-mode",
		Usage:       "logging mode",
		Value:       "prod",
	}

	if version == "development" {
		flagLogLevel.Value = "debug"
		flagLogMode.Value = "dev"
	}

	app := &cli.App{
		Name:    appName,
		Usage:   "Receive prometheus alerts via SNS and publish then to slack channel",
		Version: version,

		Flags: []cli.Flag{
			flagLogLevel,
			flagLogMode,
		},

		Before: func(ctx *cli.Context) error {
			err := setupLogger(logLevel, logFormat)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to configure the logging: %s\n", err)
			}
			return err
		},

		DefaultCommand: CommandLambda().Name,

		Commands: []*cli.Command{
			CommandLambda(),
			Debug(),
		},
	}
	defer func() {
		zap.L().Sync() //nolint:errcheck
	}()
	if err := app.Run(os.Args); err != nil {
		zap.L().Error("Failed with error", zap.Error(err))
		fmt.Printf("\n%s had failed with error:\n\n  %s\n\n", appName, err)
		os.Exit(1)
	}
}

func setupLogger(level, mode string) error {
	var config zap.Config
	switch strings.ToLower(mode) {
	case "dev":
		config = zap.NewDevelopmentConfig()
	case "prod":
		config = zap.NewProductionConfig()
	default:
		return fmt.Errorf("invalid log-mode '%s'", mode)
	}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log-level '%s': %w", level, err)
	}
	config.Level = logLevel

	l, err := config.Build()
	if err != nil {
		return fmt.Errorf("failed to build the logger: %w", err)
	}
	zap.ReplaceGlobals(l)

	return nil
}
