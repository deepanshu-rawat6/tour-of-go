BINARY := raft-kv
CMD    := ./cmd/raft-kv

.PHONY: build test lint tidy proto run-cluster clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./... -v

lint:
	go vet ./...

tidy:
	go mod tidy

proto:
	mkdir -p internal/transport/pb
	protoc --go_out=internal/transport/pb --go_opt=paths=source_relative \
	       --go-grpc_out=internal/transport/pb --go-grpc_opt=paths=source_relative \
	       -I proto proto/raft.proto

# Start a 3-node local cluster (requires 3 config files)
run-cluster: build
	@echo "Starting 3-node cluster..."
	./$(BINARY) start --config config-node1.yaml &
	./$(BINARY) start --config config-node2.yaml &
	./$(BINARY) start --config config-node3.yaml &
	@echo "Cluster started. Use 'make stop-cluster' to stop."

stop-cluster:
	pkill -f "raft-kv start" || true

clean:
	rm -f $(BINARY)
	rm -rf /tmp/raft-node{1,2,3}
