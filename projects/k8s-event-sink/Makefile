BINARY := event-sink
CMD    := ./cmd/event-sink

.PHONY: build test lint tidy clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
