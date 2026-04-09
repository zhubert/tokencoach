BINARY := tokencoach

.PHONY: build test clean release

build:
	go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

release:
	goreleaser release --clean
