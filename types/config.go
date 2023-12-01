package types

import "go.uber.org/zap"

type Config struct {
	DynamoDBName   string
	IgnoreRules    map[string]struct{}
	SlackChannel   string
	SlackChannelID string
	SlackToken     string

	Log *zap.Logger
}
