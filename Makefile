.PHONY: build run test

build:
	@go build -o bin/main 

run: build
	@./bin/main

test:
	go test -v ./...
