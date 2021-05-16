
GIT_DESCRIPTION = $(shell git describe --always --dirty --tags --long)
LINKER_FLAGS = '-s -X main.version=${GIT_DESCRIPTION}'

.PHONY: run build

# If you want,it’s possible to suppress commands from being echoed by prefixing them with the @ character.
run:
	@go run ./cmd/api

build: build-linux build-macOS

build-linux:
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/cli ./cmd/cli

build-macOS:
	mkdir -p ./bin/macOS
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/macOS/api ./cmd/api
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/macOS/cli ./cmd/cli
