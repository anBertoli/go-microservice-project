

.PHONY: run build

# If you want,it’s possible to suppress commands from being echoed by prefixing them with the @ character.
run:
	@go run ./cmd/api

build: build-linux build-macOS

git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.version=${git_description}'

build-linux:
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o ./bin/linux/api ./cmd/api
    GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o ./bin/linux/cli ./cmd/cli

build-macOS:
	mkdir -p ./bin/macOS
	GOOS=darwin GOARCH=amd64 go build -ldflags=${linker_flags} -o ./bin/macOS/api ./cmd/api
    GOOS=darwin GOARCH=amd64 go build -ldflags=${linker_flags} -o ./bin/macOS/cli ./cmd/cli

audit:
	@echo 'Tidying and verifying module dependencies...' go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

# It can still be sensible to vendor your project dependencies using the go mod vendor command.
# Vendoring dependencies in this way basically stores a complete copy of the source code for
# third-party packages in a vendor folder in your project.

# Now,when you run a command such as go run,go test or go build,the go tool will recognize the
# presence ofa vendor folder and the dependency code in the vendor folder will be used — rather
# than the code in the module cache on your local machine.
vendor:
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor




# pass argument:
# make migration name=create_example_table
example:
	@echo '"name" value: ${name}'