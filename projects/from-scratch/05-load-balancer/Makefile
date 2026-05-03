.PHONY: build test run-lb run-backend

build:
	go build ./...

test:
	go test -race ./...

run-lb:
	go run ./cmd/server/

run-backend:
	go run ./cmd/backend/
