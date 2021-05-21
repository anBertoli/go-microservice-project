#!/bin/bash

############### STORAGE ###############
STORE_DIR=/var/snapvault

sudo mkdir -p $STORE_DIR
sudo chmod 0755 $STORE_DIR

############### INSTALL CLI & MIGRATE ##############
CLI_BIN=$(ls /home/snapvault/bin/*cli*)
API_BIN=$(ls /home/snapvault/bin/*api*)

# replace cli binary
sudo rm -f /usr/local/bin/snapvault-cli
sudo cp -a $CLI_BIN /usr/local/bin/snapvault-cli

# migrate
snapvault-cli migrate \
  --database-url postgres://snapvault:${DB_PASSWORD}@localhost:5432/snapvault \
  --migrations-folder file:///home/snapvault/migrations

############### INSTALL API AS SYSTEMD UNIT ##############

sudo systemctl stop snapvault
sudo systemctl disable snapvault

# replace api binary
sudo rm -f /usr/local/bin/snapvault-api
sudo cp -a $API_BIN /usr/local/bin/snapvault-api

# replace conf file
sudo mkdir -p /etc/snapvault
sudo chmod 0755 /etc/snapvault
sudo rm -f /etc/snapvault/api.json
sudo cp -a /home/snapvault/conf/api.prod.json /etc/snapvault/api.json

# replace systemd service file
sudo rm -f /lib/systemd/system/snapvault.service
sudo cp /home/snapvault/deploy/api_conf/snapvault.service /lib/systemd/system/snapvault.service

sudo systemctl enable snapvault
sudo systemctl start snapvault