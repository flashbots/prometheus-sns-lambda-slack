package main

import (
	"errors"
	"strings"

	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/flashbots/prometheus-sns-lambda-slack/config"
	"github.com/flashbots/prometheus-sns-lambda-slack/processor"
	"github.com/urfave/cli/v2"
)

var (
	defaultSlackToken = "" // Injected at build-time
	rawIgnoreRules    = ""
)

var (
	ErrDynamoDBMissing       = errors.New("DynamoDB name must be configured")
	ErrSlackAPITokenMissing  = errors.New("Slack API token must be provided")
	ErrSlackChannelIDMissing = errors.New("Slack channel ID must be configured")
	ErrSlackChannelMissing   = errors.New("Slack channel name must be configured")
)

func CommandLambda(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "lambda",
		Usage: "Run lambda handler (default)",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Destination: &cfg.Processor.DynamoDBName,
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
				Destination: &cfg.Slack.ChannelName,
				EnvVars:     []string{"SLACK_CHANNEL_NAME"},
				Name:        "slack-channel-name",
				Usage:       "slack channel to publish the alerts to",
			},

			&cli.StringFlag{
				Destination: &cfg.Slack.ChannelID,
				EnvVars:     []string{"SLACK_CHANNEL_ID"},
				Name:        "slack-channel-id",
				Usage:       "slack channel ID to publish the alerts to",
			},

			&cli.StringFlag{
				Destination: &cfg.Slack.Token,
				EnvVars:     []string{"SLACK_TOKEN"},
				Name:        "slack-token",
				Usage:       "slack API token to be used",
			},
		},

		Before: func(_ *cli.Context) error {
			// validate inputs
			if cfg.Processor.DynamoDBName == "" {
				return ErrDynamoDBMissing
			}
			if cfg.Slack.Token == "" {
				if defaultSlackToken == "" {
					return ErrSlackAPITokenMissing
				}
				cfg.Slack.Token = defaultSlackToken
			}
			if cfg.Slack.ChannelName == "" {
				return ErrSlackChannelMissing
			}
			if cfg.Slack.ChannelID == "" {
				return ErrSlackChannelIDMissing
			}

			// parse the list of ignored rules
			for _, r := range strings.Split(rawIgnoreRules, ",") {
				if r == "" {
					continue
				}
				cfg.Processor.IgnoreRules[strings.TrimSpace(r)] = struct{}{}
			}

			return nil
		},

		Action: func(ctx *cli.Context) error {
			p, err := processor.New(cfg)
			if err != nil {
				return err
			}
			awslambda.Start(p.Lambda)
			return nil
		},
	}
}
