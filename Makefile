VERSION := $(shell git describe --tags --always --dirty="-dev" --match "v*.*.*")
VERSION := $(VERSION:v%=%)

default: build

.PHONY: build
build:
	CGO_ENABLED=0 \
	go build \
			-ldflags "-X main.version=${VERSION}" \
			-o ./bin/prometheus-sns-lambda-slack \
		github.com/flashbots/prometheus-sns-lambda-slack/cmd

.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --rm-dist

.PHONY: release
release:
	@rm -rf ./dist
	GITHUB_TOKEN=$$( gh auth token ) goreleaser release
