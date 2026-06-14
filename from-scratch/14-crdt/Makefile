.PHONY: build run test

build:
	go build -o crdt .

run: build
	./crdt

test:
	go test -v -race ./...
