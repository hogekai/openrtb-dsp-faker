.PHONY: build test run lint clean

BINARY := bin/faker
GO := go

build:
	$(GO) build -o $(BINARY) ./cmd/faker

test:
	$(GO) test -v -race ./...

run: build
	./$(BINARY)

lint:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)
