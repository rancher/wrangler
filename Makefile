all: generate validate build

generate:
	go generate

validate:
	go fmt ./...
	go vet ./...

build:
	go build ./...
