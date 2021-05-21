#!/bin/bash

############### STORAGE ###############
STORE_DIR=/var/snapvault

sudo mkdir -p $STORE_DIR
sudo chmod 0755 $STORE_DIR

############### MIGRATE  ###############
WORK_DIR=/home/snapvault

cd $WORK_DIR

CLI_BIN=$(ls bin/*cli*)
API_BIN=$(ls bin/*api*)

./${CLI_BIN} migrate \
  --database-url postgres://snapvault:snapvault-secret-password@localhost:5432/snapvault \
  --migrations-folder file://migrations

############### INSTALL API AS SYSTEMD UNIT ##############

sudo systemctl stop snapvault
sudo systemctl disable snapvault
sudo rm -f /usr/local/bin/snapvault-api
sudo rm -f /etc/snapvault/api.json
sudo rm -f /lib/systemd/system/snapvault.service

sudo cp -a $API_BIN /usr/local/bin/snapvault-api
sudo cp -a ./conf/api.prod.json /etc/snapvault/api.json

sudo cp ./deploy/api_conf/snapvault.service /lib/systemd/system/snapvault.service
sudo systemctl enable snapvault
sudo systemctl start snapvault