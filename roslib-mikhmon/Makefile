.PHONY: all build test test-race test-cover test-integration lint tidy update-golden clean

all: build test

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -tags=integration -count=1 ./test/integration/...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

update-golden:
	go test ./scripts/... -update

clean:
	rm -f coverage.out
