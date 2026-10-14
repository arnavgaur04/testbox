.PHONY: vet test build install lint

vet:
	go vet ./...

test:
	go test ./... -count=1

build:
	go build ./...

lint:
	test -z "$$(gofmt -l cmd internal pkg)"
	go vet ./...
	sh scripts/check-comments.sh

install:
	go build -o $$(go env GOPATH)/bin/testbox-tool ./cmd/core-service
	go build -o $$(go env GOPATH)/bin/testbox-api ./cmd/api-server
	@echo "Installed testbox-tool and testbox-api to $$(go env GOPATH)/bin."
