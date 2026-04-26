BINARY  := xdp-fw
CMD     := ./cmd/xdp-fw
BPF_SRC := bpf/xdp_firewall.c
BPF_OBJ := bpf/xdp_firewall.o

.PHONY: build test lint tidy generate clean

# Compile the eBPF C program (requires clang + Linux kernel headers)
generate:
	clang -O2 -target bpf -c $(BPF_SRC) -o $(BPF_OBJ)

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

# Run on Linux with root (requires generate first)
run: build
	sudo ./$(BINARY) start --config config.example.yaml

clean:
	rm -f $(BINARY) $(BPF_OBJ)
