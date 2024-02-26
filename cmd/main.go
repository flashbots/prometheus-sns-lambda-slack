package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/flashbots/prometheus-sns-lambda-slack/config"
	"github.com/flashbots/prometheus-sns-lambda-slack/logutils"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	version = "development"
)

var (
	ErrFailedToSetupLogging = errors.New("failed to setup logging")
)

const (
	appName = "prometheus-sns-lambda-slack"
)

func main() {
	cfg := &config.Config{
		Processor: config.Processor{
			IgnoreRules: make(map[string]struct{}),
		},
	}

	flagLogLevel := &cli.StringFlag{
		Destination: &cfg.Log.Level,
		EnvVars:     []string{"LOG_LEVEL"},
		Name:        "log-level",
		Usage:       "logging level",
		Value:       "info",
	}

	flagLogMode := &cli.StringFlag{
		Destination: &cfg.Log.Mode,
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
			l, err := logutils.NewLogger(&cfg.Log)
			if err != nil {
				return fmt.Errorf("%w: %w",
					ErrFailedToSetupLogging, err,
				)
			}
			zap.ReplaceGlobals(l)
			return nil
		},

		DefaultCommand: "lambda",

		Commands: []*cli.Command{
			CommandLambda(cfg),
			Debug(cfg),
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
