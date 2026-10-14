.PHONY: vet test build

vet:
	go vet ./...

test:
	go test ./... -count=1

build:
	go build ./...
