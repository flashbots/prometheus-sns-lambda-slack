# prometheus-sns-lambda-slack

Receive prometheus alerts via AWS SNS and publish then to slack channel.

## TL;DR

At the time of writing, Amazon-managed Prometheus does not support
sending alerts to Slack.

The "official" way of integration is Prometheus->SNS->Lambda->Slack
[`[1]`](https://aws.amazon.com/blogs/mt/how-to-integrate-amazon-managed-service-for-prometheus-with-slack/).

This repo is one of many possible implementations of this idea.

```shell
export SLACK_TOKEN=XXXX-NNNNNNNNNNNNN-NNNNNNNNNNNNN

./prometheus-sns-lambda-slack lambda \
  --dynamo-db-name slack-alerts \
  --slack-channel incidents \
  --slack-channel-id XXXXXXXXXXX
```

## Features

- Groups messages into threads (based on message labels).
- Deduplicates the messages (AWS managed grafana seems to be 3 instances
  in HA setup, which means that each alert sent by grafana comes as a
  triplet).
- Can filter-out alerts based on their kind.
- Flags alerts that got resolved with green check-box emoji reaction.

---

`[1]` https://aws.amazon.com/blogs/mt/how-to-integrate-amazon-managed-service-for-prometheus-with-slack/
