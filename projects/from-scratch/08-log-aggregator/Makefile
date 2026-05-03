.PHONY: build test run-aggregator run-shipper

build:
	go build ./...

test:
	go test -race ./...

run-aggregator:
	go run ./cmd/aggregator/

run-shipper:
	go run ./cmd/shipper/
