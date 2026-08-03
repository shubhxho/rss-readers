BINARY := rss-readers

.PHONY: build run vet tidy clean install

build:
	go build -o $(BINARY) .

run:
	go run .

vet:
	go vet ./...

tidy:
	go mod tidy

install:
	go install .

clean:
	rm -f $(BINARY)
