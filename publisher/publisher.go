package publisher

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/flashbots/prometheus-sns-lambda-slack/config"
	"github.com/flashbots/prometheus-sns-lambda-slack/logutils"
	"github.com/flashbots/prometheus-sns-lambda-slack/types"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

type SlackChannel struct {
	channelID   string
	channelName string
	slack       *slack.Client
}

func NewSlackChannel(cfg *config.Config) *SlackChannel {
	return &SlackChannel{
		channelName: cfg.Slack.ChannelName,
		channelID:   cfg.Slack.ChannelID,

		slack: slack.New(cfg.Slack.Token),
	}
}

func (p *SlackChannel) ChannelName() string {
	return p.channelName
}

func (p *SlackChannel) newMessage(alert *types.Alert) slack.Attachment {
	msg := slack.Attachment{}

	if alert.Status == "firing" {
		switch alert.Labels["severity"] {
		case "critical":
			msg.Color = "danger"
		case "warning":
			msg.Color = "warning"
		default:
			msg.Color = "good"
		}
	} else {
		msg.Color = "good"
	}

	msg.Title = fmt.Sprintf("%s: %s",
		strings.ToUpper(alert.Status),
		alert.Labels["alertname"],
	)

	if alertSeverity, ok := alert.Labels["severity"]; ok {
		msg.Text += fmt.Sprintf("Severity: `%s`\n", alertSeverity)
	}
	if alertSummary, ok := alert.Annotations["summary"]; ok {
		msg.Text += fmt.Sprintf("Summary: `%s`\n", alertSummary)
	}
	if alertDescription, ok := alert.Annotations["description"]; ok {
		msg.Text += fmt.Sprintf("\n%s\n\n", alertDescription)
	}
	if alertMessage, ok := alert.Annotations["message"]; ok {
		msg.Text += fmt.Sprintf("\n%s\n\n", alertMessage)
	}
	if len(alert.StartsAt) > 0 {
		msg.Text += fmt.Sprintf("Started at: `%s`\n", alert.StartsAt)
	}
	if awsAccount, ok := alert.Labels["aws_account"]; ok {
		msg.Text += fmt.Sprintf("AWS account: `%s`\n", awsAccount)
	}
	if cluster, ok := alert.Labels["cluster"]; ok {
		msg.Text += fmt.Sprintf("Kubernetes cluster: `%s`\n", cluster)
	}
	if namespace, ok := alert.Labels["namespace"]; ok {
		msg.Text += fmt.Sprintf("Kubernetes namespace: `%s`\n", namespace)
	}

	return msg
}

func (p *SlackChannel) PublishMessage(
	ctx context.Context,
	slackThreadTS string,
	alert *types.Alert,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	msg := p.newMessage(alert)
	if len(slackThreadTS) > 0 {
		if floatThreadTS, err := strconv.ParseFloat(slackThreadTS, 64); err == nil {
			sec, dec := math.Modf(floatThreadTS)
			timeSlackThreadTS := time.Unix(int64(sec), int64(dec*(1e9)))
			msg.Footer = fmt.Sprintf(
				"(follow-up to the alert published at %s)",
				timeSlackThreadTS.Format("2006-01-02T15:04:05Z07:00"),
			)
		} else {
			msg.Footer = "(follow-up)"
		}
	}

	opts := []slack.MsgOption{
		slack.MsgOptionAttachments(msg),
	}
	if len(slackThreadTS) > 0 {
		opts = append(opts,
			slack.MsgOptionTS(slackThreadTS),
		)
	}

	_, msgTS, err := p.slack.PostMessage(p.channelName, opts...)
	if err != nil {
		l.Error("Error publishing message to slack",
			zap.Error(err),
			zap.String("slack_channel", p.channelName),
			zap.String("slack_message_ts", msgTS),
			zap.String("slack_thread_ts", slackThreadTS),
		)
		return "", err
	}

	return msgTS, nil
}

func (p *SlackChannel) UpdateReaction(
	ctx context.Context,
	slackThreadTS string,
	alert *types.Alert,
) {
	l := logutils.LoggerFromContext(ctx)

	var ra, rr string
	if alert.Status == "firing" {
		ra = "rotating_light"
		rr = "white_check_mark"
	} else {
		ra = "white_check_mark"
		rr = "rotating_light"
	}

	if err := func() error {
		err := p.slack.AddReaction(ra, slack.ItemRef{
			Channel:   p.channelID,
			Timestamp: slackThreadTS,
		})
		if err == nil {
			return nil
		}
		slackErr, isSlackErr := err.(slack.SlackErrorResponse)
		if !isSlackErr {
			return err
		}
		if slackErr.Err == "already_reacted" {
			return nil
		}
		return err
	}(); err != nil {
		l.Error("Error adding reaction to slack",
			zap.Error(err),
			zap.String("slack_channel", p.channelName),
			zap.String("slack_reaction", ra),
			zap.String("slack_thread_ts", slackThreadTS),
		)
	}

	if err := func() error {
		err := p.slack.RemoveReaction(rr, slack.ItemRef{
			Channel:   p.channelID,
			Timestamp: slackThreadTS,
		})
		if err == nil {
			return nil
		}
		slackErr, isSlackErr := err.(slack.SlackErrorResponse)
		if !isSlackErr {
			return err
		}
		if slackErr.Err == "no_reaction" {
			return nil
		}
		return err
	}(); err != nil {
		l.Error("Error removing reaction from slack",
			zap.Error(err),
			zap.String("slack_channel", p.channelName),
			zap.String("slack_reaction", rr),
			zap.String("slack_thread_ts", slackThreadTS),
		)
	}
}
