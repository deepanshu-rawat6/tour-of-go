BINARY := reaper
CMD    := ./cmd/reaper

.PHONY: build test lint run-dry tidy

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

# Dry-run scan using the example config (no AWS calls succeed without real credentials,
# but this validates the binary builds and config loads correctly).
run-dry: build
	./$(BINARY) scan --config config.example.yaml --output text

clean:
	rm -f $(BINARY)
