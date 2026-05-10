.PHONY: docker-up docker-down run test integration-test

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run:
	go run ./cmd/server

test:
	go test ./...

integration-test:
	go test -tags=integration ./...
