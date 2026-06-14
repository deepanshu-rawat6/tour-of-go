.PHONY: build run test

build:
	go build -o consistent-hash .

run: build
	./consistent-hash

test:
	go test -v -race ./...
