# Makefile for ttt - terminal text editor

.PHONY: all test build run clean fmt lint

all: build

build:
	go build -ldflags="-s -w" -o bin/ttt ./cmd/ttt

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
