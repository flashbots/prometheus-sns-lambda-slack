package processor

import (
	"context"
	"errors"
	"time"

	"github.com/flashbots/prometheus-sns-lambda-slack/db"
	"github.com/flashbots/prometheus-sns-lambda-slack/logutils"
	"github.com/flashbots/prometheus-sns-lambda-slack/publisher"
	"github.com/flashbots/prometheus-sns-lambda-slack/types"
	"go.uber.org/zap"
)

var (
	ErrAlreadyLocked = errors.New("the message is already locked, let's retry later")
)

type Processor struct {
	ignoreRules map[string]struct{}

	db    *db.DB
	slack *publisher.SlackChannel
}

func New(cfg *types.Config) (*Processor, error) {
	d, err := db.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Processor{
		ignoreRules: cfg.IgnoreRules,

		db:    d,
		slack: publisher.NewSlackChannel(cfg),
	}, nil
}

func (p *Processor) ProcessMessage(
	ctx context.Context,
	topic string,
	message *types.Message,
) error {
	errs := []error{}
	for _, alert := range message.Alerts {
		for k, v := range message.CommonAnnotations {
			if _, present := alert.Annotations[k]; !present {
				alert.Annotations[k] = v
			}
		}
		for k, v := range message.CommonLabels {
			if _, present := alert.Labels[k]; !present {
				alert.Labels[k] = v
			}
		}
		_timestamp, err := time.Parse("2006-01-02T15:04:05Z07:00", alert.StartsAt) // Grafana
		if err != nil {
			_timestamp, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", alert.StartsAt) // Prometheus
		}
		if err == nil {
			alert.StartsAt = _timestamp.Format("2006-01-02T15:04:05Z07:00")
		}
		if err := p.processAlert(ctx, topic, &alert); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (p *Processor) processAlert(
	ctx context.Context,
	topic string,
	alert *types.Alert,
) (err error) {
	l := logutils.LoggerFromContext(ctx).With(
		zap.String("alert_fingerprint", alert.Fingerprint()),
		zap.String("alert_labels_fingerprint", alert.LabelsFingerprint()),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	slackMessageID := p.slackMessageID(alert)
	slackThreadID := p.slackThreadID(alert)
	slackThreadTS := ""

	if _, ignore := p.ignoreRules[alert.Labels["alertname"]]; ignore {
		l.Info("Skipped the alert according to ignore-rules configuration",
			zap.Any("alert", alert),
		)
	}

	// whatever the issues with DB we will try to publish to slack at least once
	shouldPublish := true
	defer func() {
		if shouldPublish {
			_, err2 := p.slack.PublishMessage(ctx, slackThreadTS, alert)
			if err2 == nil {
				l.Warn("Emergency-published alert",
					zap.Any("alert", alert),
				)
			}
			err = errors.Join(err, err2)
		}
	}()

	slackMessageTS, err := p.db.GetSlackMessageTS(ctx, topic, slackMessageID)
	if err != nil {
		return err
	}
	if len(slackMessageTS) > 0 {
		// already published
		shouldPublish = false
		return nil
	}
	didLock, err := p.db.LockSlackMessage(ctx, topic, slackMessageID)
	if !didLock && err == nil {
		// another grafana's HA instance is about to publish
		shouldPublish = false
		return ErrAlreadyLocked
	}

	slackThreadTS, err = p.db.GetSlackThreadTS(ctx, topic, slackThreadID)
	if err != nil {
		return err
	}

	slackMessageTS, err = p.slack.PublishMessage(ctx, slackThreadTS, alert)
	if err != nil {
		return err
	}
	shouldPublish = false
	l.Info("Published alert",
		zap.Any("alert", alert),
	)

	if len(slackThreadTS) == 0 {
		slackThreadTS = slackMessageTS
		// we published to slack, we can ignore errors here
		_ = p.db.SetSlackThreadTS(ctx, topic, slackThreadID, slackThreadTS)
	}

	if len(slackThreadTS) > 0 {
		p.slack.UpdateReaction(ctx, slackThreadTS, alert)
	}

	return nil
}

func (p *Processor) slackThreadID(alert *types.Alert) string {
	return "alert/" + p.slack.ChannelName() + "/" + alert.LabelsFingerprint()
}

func (p *Processor) slackMessageID(alert *types.Alert) string {
	return "message/" + p.slack.ChannelName() + "/" + alert.Fingerprint()
}
