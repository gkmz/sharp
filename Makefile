.PHONY: test run build fmt fmt-check vet verify release-check snapshot tidy

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

fmt-check:
	@test -z "$$(gofmt -l cmd internal pkg)" || { gofmt -l cmd internal pkg; exit 1; }

vet:
	go vet ./...

verify: fmt-check vet test build

release-check:
	goreleaser check

snapshot:
	goreleaser release --snapshot --clean
