.PHONY: test build vet fmt

build:
	go build -o git-declutter -ldflags "-X main.version=0.1.0-dev" .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
