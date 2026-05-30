.PHONY: build run test

build:
	go build -o saga-orchestrator ./cmd

run: build
	./saga-orchestrator

test:
	go test -v -race ./...
