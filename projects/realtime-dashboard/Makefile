.PHONY: build run test

build:
	go build ./...

run:
	SCHEDULER_URL=http://localhost:8080 go run ./cmd/dashboard/

test:
	go test ./...
