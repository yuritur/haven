
.PHONY: build test lint fmt vet tidy clean

build:
	go build -o haven ./cmd/haven/

test:
	go test -race ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f haven
