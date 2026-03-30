.PHONY: build test test-race docker-up docker-down run

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run:
	CONFIG_PATH=configs/config.yaml go run ./cmd/scheduler/
