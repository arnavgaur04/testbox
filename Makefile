.PHONY: vet test build install

vet:
	go vet ./...

test:
	go test ./... -count=1

build:
	go build ./...

install:
	go build -o $$(go env GOPATH)/bin/testbox-tool ./cmd/core-service
	go build -o $$(go env GOPATH)/bin/testbox-api ./cmd/api-server
	@echo "Installed testbox-tool and testbox-api to $$(go env GOPATH)/bin."
