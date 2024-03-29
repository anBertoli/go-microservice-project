
GIT_DESCRIPTION = $(shell git describe --always --dirty --tags --long)
LINKER_FLAGS = '-s -X main.version=${GIT_DESCRIPTION}'

.PHONY: run build build-linux build-mac clean remote/provisioning remote/deploy cloc

build: build-linux build-mac

build-linux: clean
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/snapvault-api_${GIT_DESCRIPTION} ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/snapvault-cli_${GIT_DESCRIPTION} ./cmd/cli

build-mac: clean
	mkdir -p ./bin/mac
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/mac/snapvault-api_${GIT_DESCRIPTION} ./cmd/api
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/mac/snapvault-cli_${GIT_DESCRIPTION} ./cmd/cli

remote/provisioning:
	@if [ -z ${REMOTE_IP} ]; then\
		echo "error: remote IP not set"; \
		exit 1; \
	fi
	scp -i ~/.ssh/hetzner_rsa  -r ./deploy root@${REMOTE_IP}:/root/
	ssh -i ~/.ssh/hetzner_rsa  root@${REMOTE_IP} "chmod +0700 /root/deploy/prep.sh"
	ssh -i ~/.ssh/hetzner_rsa  root@${REMOTE_IP} "/root/deploy/prep.sh"
	ssh -i ~/.ssh/hetzner_rsa  root@${REMOTE_IP} "rm -rf /root/deploy"
	ssh -i ~/.ssh/hetzner_rsa  snapvault@${REMOTE_IP}

remote/deploy: build-linux
	@if [ -z ${REMOTE_IP} ]; then\
		echo "error: remote IP not set"; \
		exit 1; \
	fi
	ssh -t -i ~/.ssh/hetzner_rsa  snapvault@${REMOTE_IP} "rm -rf /home/snapvault/deploy /home/snapvault/bin /home/snapvault/migrations /home/snapvault/conf"
	scp -i ~/.ssh/hetzner_rsa -r ./bin/linux/ snapvault@${REMOTE_IP}:/home/snapvault/bin
	scp -i ~/.ssh/hetzner_rsa -r ./deploy/ snapvault@${REMOTE_IP}:/home/snapvault/deploy
	scp -i ~/.ssh/hetzner_rsa -r ./migrations/ snapvault@${REMOTE_IP}:/home/snapvault/migrations
	scp -i ~/.ssh/hetzner_rsa -r ./conf/ snapvault@${REMOTE_IP}:/home/snapvault/conf
	ssh -t -i ~/.ssh/hetzner_rsa  snapvault@${REMOTE_IP} "chmod +0700 /home/snapvault/deploy/deploy.sh"
	ssh -t -i ~/.ssh/hetzner_rsa  snapvault@${REMOTE_IP} "/home/snapvault/deploy/deploy.sh"

clean:
	rm -rf bin

cloc:
	cloc pkg/ services/ cmd/ deploy/ conf/ migrations/ .gitignore makefile README.md go.mod go.sum  --md