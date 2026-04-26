BINARY := gocker
CMD    := ./cmd/gocker

.PHONY: build test lint tidy clean

# Cross-compile for Linux (required — uses Linux syscalls)
build:
	GOOS=linux go build -o $(BINARY) $(CMD)

# Unit tests: image/store package runs on any OS; Linux-tagged packages need GOOS=linux
test:
	go test ./internal/image/... -v
	@echo "Note: cgroup, overlay, namespace, container tests require Linux + root."
	@echo "      Run 'sudo go test ./... -tags linux' on a Linux host."

lint:
	GOOS=linux go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)

# Full integration tests — Linux + root only
test-integration:
	sudo go test ./... -tags linux -run Integration -v
