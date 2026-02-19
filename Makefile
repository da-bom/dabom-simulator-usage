.PHONY: build run test clean vet

BINARY=simulator-usage
CMD=./cmd/simulator

build:
	go build -ldflags="-s -w" -o $(BINARY) $(CMD)

run: build
	./$(BINARY) --config configs/config.yaml

run-dev: build
	./$(BINARY) --config configs/config.dev.yaml

test:
	go test ./...

test-v:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

vet:
	go vet ./...

clean:
	rm -f $(BINARY) coverage.out
