.PHONY: build test docker-up docker-down run-producer run-consumer

build:
	go build ./...

test:
	go test -race ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run-producer:
	go run ./cmd/producer/

run-consumer:
	go run ./cmd/consumer/
