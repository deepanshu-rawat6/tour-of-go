BINARY := detect
CMD    := ./cmd/detect

.PHONY: build test lint tidy run-once clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./... -v

lint:
	go vet ./...

tidy:
	go mod tidy

run-once: build
	./$(BINARY) run --config config.example.yaml

clean:
	rm -f $(BINARY)
