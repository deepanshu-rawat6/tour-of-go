.PHONY: build run test

build:
	go build -o database-internals .

run: build
	./database-internals

test:
	go test -v -race ./...
