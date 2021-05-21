#!/bin/bash

USERNAME=snapvault
DB_PASSWORD=snapvault-secret-password

apt update
apt --yes upgrade



############### Create new user ###############
useradd --create-home --shell "/bin/bash" --groups sudo "${USERNAME}"
passwd --delete "${USERNAME}"
chage --lastday 0 "${USERNAME}"

# Copy the SSH keys from the root user to the new user
# change ownership of the directory and files
cp -r /root/.ssh /home/${USERNAME}
chown -R ${USERNAME}:${USERNAME} /home/${USERNAME}



############### Firewall ###############
# Configure the firewall to allow SSH, HTTP and HTTPS traffic.
ufw allow 22
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 4000/tcp # testing purpose only
ufw --force enable



############### Postgres ###############
apt-get install postgresql -y

# Set up the snapvault DB and create a user account with password login.
sudo -i -u postgres psql -c "CREATE DATABASE ${USERNAME}"
sudo -i -u postgres psql -d ${USERNAME} -c "CREATE ROLE ${USERNAME} WITH LOGIN PASSWORD '${DB_PASSWORD}'"

systemctl restart postgresql



############### Nginx ###############
WORK_DIR=/usr/src

sudo apt-get install -y build-essential
sudo apt-get install -y libpcre3 libpcre3-dev zlib1g zlib1g-dev libssl-dev

# download nginx, decompress it, compile it and install it
cd $WORK_DIR
wget -O "$WORK_DIR/nginx-1.19.6.tar.gz" http://nginx.org/download/nginx-1.19.6.tar.gz
tar -zxvf nginx-1.19.6.tar.gz
cd nginx-1.19.6

./configure \
  --sbin-path=/usr/sbin/nginx --conf-path=/etc/nginx/nginx.conf \
  --error-log-path=/var/log/nginx/error.log --http-log-path=/var/log/nginx/access.log \
  --with-pcre --pid-path=/var/run/nginx.pid --with-http_ssl_module
make
make install

# prepare log directories for both the generic server block
# and specific for snap vault (nginx) logs
mkdir -p /var/log/nginx
mkdir -p /var/log/nginx/go-gallery/

# create configuration files specific for snap-vault
# where nginx server blocks will be put
mkdir -p /etc/nginx/sites/
