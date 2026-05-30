.PHONY: build run test

build:
	go build -o bloom-filter .

run: build
	./bloom-filter

test:
	go test -v -race ./...
