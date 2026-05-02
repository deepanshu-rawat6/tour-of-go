.PHONY: build test docker-up docker-down run

build:
	go build ./...

test:
	go test -race ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run:
	go run ./cmd/server/
