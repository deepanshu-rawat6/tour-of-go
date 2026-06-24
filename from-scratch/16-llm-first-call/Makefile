build:
	go build -o llm-first-call .

run: build
	./llm-first-call

server: build
	./llm-first-call server

test:
	go test -race ./...
