# Makefile for ttt - terminal text editor

.PHONY: all test build run clean fmt lint chaos chaos-docker chaos-docker-build

all: build

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/ttt ./cmd/ttt

test:
	go test ./...

run: build
	./bin/ttt

fmt:
	gofmt -w .

lint:
	golint ./...

chaos:
	go test -v -tags chaos -count=1 ./tests/chaos/ -run TestChaosMonkey

chaos-docker-build:
	docker build -t ttt-chaos -f tests/chaos/Dockerfile .

chaos-docker:
	mkdir -p chaos-output
	docker run --rm -v $(PWD)/chaos-output:/output ttt-chaos

clean:
	rm -rf bin/
	find . -name '*.test' -delete
