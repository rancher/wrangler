all: validate build

validate:
	go fmt ./...
	go vet ./...

build:
	go build ./...
test:
	bash scripts/test.sh
