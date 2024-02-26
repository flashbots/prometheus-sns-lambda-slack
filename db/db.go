package db

import (
	"context"
	"fmt"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/flashbots/prometheus-sns-lambda-slack/logutils"
	"go.uber.org/zap"
)

const (
	attrExpireOn       = "expire_on"
	attrID             = "id"
	attrSlackMessageTS = "slack_message_ts"
	attrSlackThreadTS  = "slack_thread_ts"
	attrSNSTopic       = "sns_topic"

	lockTimeout               = time.Second
	slackMessageExpiryTimeout = time.Minute
	slackThreadExpiryTimeout  = 30 * 24 * time.Hour
)

type DB struct {
	client *dynamodb.DynamoDB
	name   string
}

func New(name string) (*DB, error) {
	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return &DB{
		client: dynamodb.New(s),
		name:   name,
	}, nil
}

func (db *DB) LockSlackThread(
	ctx context.Context,
	topic string,
	slackThreadID string,
) (bool, error) {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	input := &dynamodb.PutItemInput{
		TableName: aws.String(db.name),

		Item: map[string]*dynamodb.AttributeValue{
			attrID:       {S: aws.String(slackThreadID)},
			attrSNSTopic: {S: &topic},

			attrExpireOn: {N: aws.String(fmt.Sprintf("%d",
				time.Now().Add(lockTimeout).Unix(),
			))},
		},

		ConditionExpression:      aws.String("attribute_not_exists(#id)"),
		ExpressionAttributeNames: map[string]*string{"#id": aws.String(attrID)},
	}
	output, err := db.client.PutItemWithContext(ctx, input)

	if err == nil {
		return true, nil
	}
	if _, didCndChkFail := err.(*dynamodb.ConditionalCheckFailedException); didCndChkFail {
		return false, nil
	}

	l.Error("Failed to lock the slack thread",
		zap.Any("input", input),
		zap.Any("output", output),
		zap.Error(err),
	)

	return false, err
}

func (db *DB) GetSlackThreadTS(
	ctx context.Context,
	topic string,
	slackThreadID string,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	input := &dynamodb.GetItemInput{
		TableName: aws.String(db.name),

		Key: map[string]*dynamodb.AttributeValue{
			attrSNSTopic: {S: aws.String(topic)},
			attrID:       {S: aws.String(slackThreadID)},
		},
	}

	output, err := db.client.GetItemWithContext(ctx, input)
	if err != nil {
		l.Error("Failed to get slack thread timestamp",
			zap.Any("input", input),
			zap.Any("output", output),
			zap.Error(err),
		)
		return "", err
	}

	if len(output.Item) == 0 {
		return "", nil
	}

	ts, ok := output.Item[attrSlackThreadTS]
	if !ok {
		return "", nil
	}

	return *ts.S, nil
}

func (db DB) SetSlackThreadTS(
	ctx context.Context,
	topic string,
	slackThreadID string,
	slackThreadTS string,
) error {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	input := &dynamodb.PutItemInput{
		TableName: aws.String(db.name),

		Item: map[string]*dynamodb.AttributeValue{
			attrID:            {S: aws.String(slackThreadID)},
			attrSlackThreadTS: {S: aws.String(slackThreadTS)},
			attrSNSTopic:      {S: aws.String(topic)},

			attrExpireOn: {N: aws.String(fmt.Sprintf("%d",
				time.Now().Add(slackThreadExpiryTimeout).Unix(),
			))},
		},
	}
	output, err := db.client.PutItemWithContext(ctx, input)
	if err != nil {
		l.Error("Failed to set slack thread timestamp",
			zap.Any("input", input),
			zap.Any("output", output),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (db *DB) LockSlackMessage(
	ctx context.Context,
	topic string,
	slackMessageID string,
) (bool, error) {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	input := &dynamodb.PutItemInput{
		TableName: aws.String(db.name),

		Item: map[string]*dynamodb.AttributeValue{
			attrID:       {S: aws.String(slackMessageID)},
			attrSNSTopic: {S: aws.String(topic)},

			attrExpireOn: {N: aws.String(fmt.Sprintf("%d",
				time.Now().Add(lockTimeout).Unix(),
			))},
		},

		ConditionExpression:      aws.String("attribute_not_exists(#id)"),
		ExpressionAttributeNames: map[string]*string{"#id": aws.String(attrID)},
	}
	output, err := db.client.PutItemWithContext(ctx, input)

	if err == nil {
		return true, nil
	}
	if _, isCndChkFailedExc := err.(*dynamodb.ConditionalCheckFailedException); isCndChkFailedExc {
		return false, nil
	}

	l.Error("Failed to lock the slack message",
		zap.Any("input", input),
		zap.Any("output", output),
		zap.Error(err),
	)

	return false, err
}

func (db *DB) GetSlackMessageTS(
	ctx context.Context,
	topic string,
	slackMessageID string,
) (string, error) {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	input := &dynamodb.GetItemInput{
		TableName: aws.String(db.name),

		Key: map[string]*dynamodb.AttributeValue{
			attrSNSTopic: {S: aws.String(topic)},
			attrID:       {S: aws.String(slackMessageID)},
		},
	}

	output, err := db.client.GetItemWithContext(ctx, input)
	if err != nil {
		l.Error("Failed to get slack message timestamp",
			zap.Any("input", input),
			zap.Any("output", output),
			zap.Error(err),
		)
		return "", err
	}

	if len(output.Item) == 0 {
		return "", nil
	}

	ts, ok := output.Item[attrSlackMessageTS]
	if !ok {
		return "", nil
	}

	return *ts.S, nil
}

func (db *DB) SetSlackMessageTS(
	ctx context.Context,
	topic string,
	slackMessageID string,
	slackMessageTS string,
) error {
	l := logutils.LoggerFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	input := &dynamodb.PutItemInput{
		TableName: aws.String(db.name),

		Item: map[string]*dynamodb.AttributeValue{
			attrID:             {S: aws.String(slackMessageID)},
			attrSlackMessageTS: {S: aws.String(slackMessageTS)},
			attrSNSTopic:       {S: aws.String(topic)},

			attrExpireOn: {N: aws.String(fmt.Sprintf("%d",
				time.Now().Add(slackThreadExpiryTimeout).Unix(),
			))},
		},
	}
	output, err := db.client.PutItemWithContext(ctx, input)
	if err != nil {
		l.Error("Failed to set slack thread timestamp",
			zap.Any("input", input),
			zap.Any("output", output),
			zap.Error(err),
		)
		return err
	}
	return nil
}
