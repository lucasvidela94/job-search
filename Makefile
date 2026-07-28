BINARY=jobsearch
MODULE=github.com/lucasvidela94/jobsearch

.PHONY: build test clean fmt vet run

build:
	go build -o $(BINARY) ./cmd/$(BINARY)

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY) coverage.out coverage.html

run:
	go run ./cmd/$(BINARY) $(ARGS)
