package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"os"

	"github.com/flashbots/prometheus-sns-lambda-slack/config"
	"github.com/flashbots/prometheus-sns-lambda-slack/processor"
	"github.com/flashbots/prometheus-sns-lambda-slack/types"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	snsTopicARN string
)

func Debug(cfg *config.Config) *cli.Command {
	base := CommandLambda(cfg)

	return &cli.Command{
		Name:  "debug",
		Usage: "Manually process lambda message (for example to debug slack token's scopes and permissions)",

		Flags: append(base.Flags, []cli.Flag{
			&cli.StringFlag{
				Destination: &snsTopicARN,
				EnvVars:     []string{"SNS_TOPIC_ARN"},
				Name:        "sns-topic-arn",
				Usage:       "the ARN of the SNS topic to mimic",
			},
		}...),

		ArgsUsage: "/path/to/file/with/message.json",

		Before: func(clictx *cli.Context) error {
			if err := base.Before(clictx); err != nil {
				return err
			}
			if snsTopicARN == "" {
				return errors.New("The ARN of SNS topic to mimic must be provided")
			}
			if !clictx.Args().Present() {
				return errors.New("Path to the message file is missing")
			}
			return nil
		},

		Action: func(clictx *cli.Context) error {
			l := zap.L().With(
				zap.String("event_id", uuid.New().String()),
			)
			defer l.Sync() //nolint:errcheck

			bytes, err := os.ReadFile(clictx.Args().First())
			if err != nil {
				return err
			}
			var m types.Message
			if err := json.Unmarshal(bytes, &m); err != nil {
				return err
			}

			ignoreRulesSet := make(map[string]struct{})
			for _, r := range strings.Split(rawIgnoreRules, ",") {
				ignoreRulesSet[strings.TrimSpace(r)] = struct{}{}
			}

			p, err := processor.New(cfg)
			if err != nil {
				return err
			}

			return p.ProcessMessage(context.Background(), snsTopicARN, &m)
		},
	}
}
