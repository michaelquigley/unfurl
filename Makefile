.PHONY: build test clean

build:
	go install ./cmd/unfurl

test:
	go test ./... -count=1
	go vet ./...

clean:
	go clean
