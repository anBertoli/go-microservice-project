
GIT_DESCRIPTION = $(shell git describe --always --dirty --tags --long)
LINKER_FLAGS = '-s -X main.version=${GIT_DESCRIPTION}'
REMOTE_IP = '168.119.170.168'

.PHONY: run build

# If you want,it’s possible to suppress commands from being echoed by prefixing them with the @ character.
run:
	go run ./cmd/api

build: build-linux build-mac

build-linux:
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/cli ./cmd/cli

build-mac:
	mkdir -p ./bin/macOS
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/mac/api ./cmd/api
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/mac/cli ./cmd/cli

remote-ssh:
	ssh snapvault@${REMOTE_IP}
