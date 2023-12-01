package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/flashbots/prometheus-sns-lambda-slack/processor"
	"github.com/flashbots/prometheus-sns-lambda-slack/types"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	defaultSlackToken = "" // Injected at build-time
	rawIgnoreRules    = ""

	cfg = types.Config{
		IgnoreRules: make(map[string]struct{}),
	}
)

var (
	ErrDynamoDBMissing       = errors.New("DynamoDB name must be configured")
	ErrSlackAPITokenMissing  = errors.New("Slack API token must be provided")
	ErrSlackChannelIDMissing = errors.New("Slack channel ID must be configured")
	ErrSlackChannelMissing   = errors.New("Slack channel name must be configured")
)

func CommandLambda() *cli.Command {
	return &cli.Command{
		Name:  "lambda",
		Usage: "Run lambda handler (default)",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Destination: &cfg.DynamoDBName,
				EnvVars:     []string{"DYNAMODB_NAME"},
				Name:        "dynamo-db-name",
				Usage:       "the name of Dynamo DB to keep the track of alerts",
			},

			&cli.StringFlag{
				Destination: &rawIgnoreRules,
				EnvVars:     []string{"IGNORE_RULES"},
				Name:        "ignore-rules",
				Usage:       "comma-separated list of rules to ignore",
			},

			&cli.StringFlag{
				Destination: &cfg.SlackChannel,
				EnvVars:     []string{"SLACK_CHANNEL"},
				Name:        "slack-channel",
				Usage:       "slack channel to publish the alerts to",
			},

			&cli.StringFlag{
				Destination: &cfg.SlackChannelID,
				EnvVars:     []string{"SLACK_CHANNEL_ID"},
				Name:        "slack-channel-id",
				Usage:       "slack channel ID to publish the alerts to",
			},

			&cli.StringFlag{
				Destination: &cfg.SlackToken,
				EnvVars:     []string{"SLACK_TOKEN"},
				Name:        "slack-token",
				Usage:       "slack API token to be used",
			},
		},

		Before: func(ctx *cli.Context) error {
			if cfg.DynamoDBName == "" {
				return ErrDynamoDBMissing
			}
			if cfg.SlackToken == "" {
				if defaultSlackToken == "" {
					return ErrSlackAPITokenMissing
				}
				cfg.SlackToken = defaultSlackToken
			}
			if cfg.SlackChannel == "" {
				return ErrSlackChannelMissing
			}
			if cfg.SlackChannelID == "" {
				return ErrSlackChannelIDMissing
			}
			return nil
		},

		Action: func(ctx *cli.Context) error {
			lambda.Start(Lambda)
			return nil
		},
	}
}

func Lambda(clictx context.Context, event events.SNSEvent) error {
	l := zap.L().With(
		zap.String("event_id", uuid.New().String()),
	)
	defer l.Sync() //nolint:errcheck
	cfg.Log = l

	for _, r := range strings.Split(rawIgnoreRules, ",") {
		if r == "" {
			continue
		}
		cfg.IgnoreRules[strings.TrimSpace(r)] = struct{}{}
	}

	p, err := processor.New(&cfg)
	if err != nil {
		return err
	}

	errs := []error{}
	for _, r := range event.Records {
		var m types.Message
		if err := json.Unmarshal([]byte(r.SNS.Message), &m); err != nil {
			l.Error("Error un-marshalling message",
				zap.String("message", strings.Replace(r.SNS.Message, "\n", " ", -1)),
				zap.Error(err),
			)
			errs = append(errs, err)
			continue
		}
		if err := p.ProcessMessage(clictx, r.SNS.TopicArn, &m); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}
	return nil
}
