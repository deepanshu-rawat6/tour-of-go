build:
	go build -ldflags="-X main.version=v0.1.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo none)" -o devctl .

run: build
	./devctl

install: build
	mv devctl /usr/local/bin/devctl

completion-bash: build
	./devctl completion bash > /tmp/devctl-completion.bash
	source /tmp/devctl-completion.bash

test:
	go test -race ./...
