.PHONY: build run test produce consume clean

build:
	go build -o commit-log .

run: build
	./commit-log

test:
	go test -v -race ./...

produce:
	@echo "PRODUCE orders 0 hello-world" | nc localhost 9092

consume:
	@echo "CONSUME orders 0 0" | nc localhost 9092

clean:
	rm -rf ./data commit-log
