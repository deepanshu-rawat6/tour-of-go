.PHONY: build run escape-analysis

build:
	go build -o production-go .

run: build
	./production-go

escape-analysis:
	go build -gcflags='-m -m' ./... 2>&1 | head -50
