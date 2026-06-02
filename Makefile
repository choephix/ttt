# Makefile for ttt - terminal text editor

.PHONY: all test build run clean fmt lint

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

clean:
	rm -rf bin/
	find . -name '*.test' -delete
