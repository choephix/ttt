# Makefile for Go Terminal Editor project

.PHONY: all test build run clean fmt lint

all: build

build:
	go build -o bin/pico ./cmd/pico

test:
	go test ./...

run: build
	./bin/pico

fmt:
	gofmt -w .

lint:
	golint ./...

clean:
	rm -rf bin/
	find . -name '*.test' -delete
