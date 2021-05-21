
GIT_DESCRIPTION = $(shell git describe --always --dirty --tags --long)
LINKER_FLAGS = '-s -X main.version=${GIT_DESCRIPTION}'
REMOTE_IP = '168.119.170.168'

.PHONY: run build clean

# If you want,it’s possible to suppress commands from being echoed by prefixing them with the @ character.
run:
	go run ./cmd/api

build: build-linux build-mac

build-linux:
	mkdir -p ./bin/linux
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/snapvault-api_${GIT_DESCRIPTION} ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/linux/snapvault-cli_${GIT_DESCRIPTION} ./cmd/cli

build-mac:
	mkdir -p ./bin/mac
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/mac/snapvault-api_${GIT_DESCRIPTION} ./cmd/api
	GOOS=darwin GOARCH=amd64 go build -ldflags=${LINKER_FLAGS} -o ./bin/mac/snapvault-cli_${GIT_DESCRIPTION} ./cmd/cli

remote/provisioning:
	scp -i ~/.ssh/hetzner_rsa ./deploy/prep.sh root@${REMOTE_IP}:/root/snapvault_prep.sh
	ssh -i ~/.ssh/hetzner_rsa  root@${REMOTE_IP} "chmod 0700 /root/snapvault_prep.sh"
	ssh -i ~/.ssh/hetzner_rsa  root@${REMOTE_IP} "/root/snapvault_prep.sh"
	ssh -i ~/.ssh/hetzner_rsa  root@${REMOTE_IP} "rm /root/snapvault_prep.sh"

remote/deploy:
	scp -i ~/.ssh/hetzner_rsa -r ./bin/linux root@${REMOTE_IP}:/home/snapvault/bin
	scp -i ~/.ssh/hetzner_rsa -r ./deploy root@${REMOTE_IP}:/home/snapvault/deploy
	scp -i ~/.ssh/hetzner_rsa -r ./migrations root@${REMOTE_IP}:/home/snapvault/migrations
	scp -i ~/.ssh/hetzner_rsa -r ./conf root@${REMOTE_IP}:/home/snapvault/conf


remote/upload:
	ssh root@${REMOTE_IP} "mkdir -p /home/snapvault"
	scp -i ~/.ssh/hetzner_rsa -r ./bin/linux root@${REMOTE_IP}:/home/snapvault/bin
	scp -i ~/.ssh/hetzner_rsa -r ./deploy root@${REMOTE_IP}:/home/snapvault/deploy
	scp -i ~/.ssh/hetzner_rsa -r ./migrations root@${REMOTE_IP}:/home/snapvault/migrations
	scp -i ~/.ssh/hetzner_rsa -r ./conf root@${REMOTE_IP}:/home/snapvault/conf
	ssh -i ~/.ssh/hetzner_rsa root@${REMOTE_IP} "chown snapvault -R /home/snapvault"

clean:
	rm -rf bin
