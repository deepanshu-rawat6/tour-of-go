.PHONY: build run test

build:
	go build -o event-sourced-ledger ./cmd

run: build
	./event-sourced-ledger

test:
	go test -v -race ./...
