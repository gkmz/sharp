.PHONY: test run build fmt tidy

test:
	go test ./...

run:
	go run ./cmd/sharp

build:
	go build ./cmd/sharp

fmt:
	gofmt -w cmd internal pkg

tidy:
	go mod tidy

