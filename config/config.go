package config

type Config struct {
	Log       Log
	Processor Processor
	Slack     Slack
}

type Log struct {
	Level string
	Mode  string
}

type Processor struct {
	DynamoDBName string
	IgnoreRules  map[string]struct{}
}

type Slack struct {
	ChannelID   string
	ChannelName string
	Token       string
}
