.PHONY: build test run-server run-producer run-consumer

build:
	go build ./...

test:
	go test -race ./...

run-server:
	go run ./cmd/server/

run-producer:
	go run ./cmd/producer/

run-consumer:
	go run ./cmd/consumer/
