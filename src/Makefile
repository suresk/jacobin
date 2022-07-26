all: build

compile:
	go build -ldflags "-X 'jacobin/globals.Branch="`git rev-parse --abbrev-ref HEAD`"' -X 'jacobin/globals.Commit="`git rev-parse --short HEAD`"'"

build: test compile

test:
	go test -short ./...

test-all:
	go test ./...

release: test-all compile
