.PHONY: build test cover run certs

build:
	go build ./...

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

run:
	go run ./cmd/server/

certs:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/ca.key -out certs/ca.crt -days 365 -nodes -subj "/CN=ca"
	openssl req -newkey rsa:4096 -keyout certs/server.key -out certs/server.csr -nodes -subj "/CN=localhost"
	openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 365
	openssl req -newkey rsa:4096 -keyout certs/client.key -out certs/client.csr -nodes -subj "/CN=client"
	openssl x509 -req -in certs/client.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/client.crt -days 365
