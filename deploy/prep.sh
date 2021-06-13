#!/bin/bash

USERNAME=snapvault

apt update
apt --yes upgrade



############### CREATE NEW USER ###############
useradd --create-home --shell "/bin/bash" --groups sudo "${USERNAME}"
passwd --delete "${USERNAME}"
chage --lastday 0 "${USERNAME}"

# Copy the SSH keys from the root user to the new user
# change ownership of the directory and files
cp -r /root/.ssh /home/${USERNAME}
chown -R ${USERNAME}:${USERNAME} /home/${USERNAME}



############### FIREWALL ###############
# Configure the firewall to allow SSH, HTTP and HTTPS traffic.
ufw allow 22
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable



############### POSTGRES ###############
apt-get install postgresql -y

# Set up the snapvault DB and create a user account with password login.
sudo -i -u postgres psql -c "CREATE DATABASE ${USERNAME}"
sudo -i -u postgres psql -d ${USERNAME} -c "CREATE ROLE ${USERNAME} WITH LOGIN PASSWORD '${DB_PASSWORD}'"

systemctl restart postgresql



############### NGINX ###############
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
# and for the specific application (snap vault)
mkdir -p /var/log/nginx
mkdir -p /var/log/nginx/snapvault/
touch /var/log/nginx/snapvault/error.log
touch /var/log/nginx/snapvault/access.log

# create log files for grafana proxying
mkdir -p /var/log/nginx/grafana/
touch /var/log/nginx/grafana/access.log
touch /var/log/nginx/grafana/error.log

# copy nginx confs into nginx folder
mkdir -p /etc/nginx/sites/
cp /root/deploy/nginx_conf/nginx.conf /etc/nginx/nginx.conf
cp /root/deploy/nginx_conf/nginx.snapvault.conf /etc/nginx/sites/nginx.snapvault.conf



############### HTTPS ###############

CERT=/etc/letsencrypt/live/snapvault.ablab.dev/fullchain.pem
PRIV=/etc/letsencrypt/live/snapvault.ablab.dev/privkey.pem

if [[ -f "$CERT" && -f "$PRIV" ]]; then
  echo "Cert files exist, skipping HTTPS certificates creation."
else
  echo "Cert files doesn't exist, creating new HTTPS certificates."
  apt install snapd -y

  sudo snap install core
  sudo snap refresh core
  sudo apt-get remove certbot

  sudo snap install --classic certbot
  rm -rf /usr/bin/certbot
  sudo ln -s /snap/bin/certbot /usr/bin/certbot

  sudo certbot certonly --standalone -m andrea.bertpp@gmail.com -d snapvault.ablab.dev -d www.snapvault.ablab.dev --agree-tos -n
fi



############### NGINX (SYSTEMD UNIT) ###############

cp /root/deploy/nginx_conf/nginx.service /lib/systemd/system/nginx.service
systemctl enable nginx
systemctl start nginx



############### PROMETHEUS ###############
cd $WORK_DIR

# Download prometheus binary for Linux.
wget -O ./prometheus-2.27.1.tar.gz https://github.com/prometheus/prometheus/releases/download/v2.27.1/prometheus-2.27.1.linux-amd64.tar.gz
tar -zxvf prometheus-2.27.1.tar.gz
cp prometheus-2.27.1.linux-amd64/prometheus /usr/local/bin/prometheus

# Prep configuration file.
mkdir -p /etc/prometheus
cp /root/deploy/prom_conf/prometheus.conf.yaml /etc/prometheus/prometheus.conf.yaml

# Create storage directory.
mkdir -p /var/lib/prometheus/



############### PROMETHEUS (SYSTEMD UNIT) ###############
cp /root/deploy/prom_conf/prometheus.service /lib/systemd/system/prometheus.service
systemctl enable prometheus
systemctl start prometheus



############### GRAFANA ###############
cd $WORK_DIR

sudo apt-get install -y adduser libfontconfig1
sudo wget https://dl.grafana.com/oss/release/grafana_8.0.1_amd64.deb
sudo dpkg -i grafana_8.0.1_amd64.deb

# Adjust configuration file
sed -i "s/^;root_url.*$/root_url = https:\/\/snapvault.ablab.dev\/grafana/g" /etc/grafana/grafana.ini
sed -i "s/^;serve_from_sub_path = false.*$/serve_from_sub_path = true/g" /etc/grafana/grafana.ini



############### GRAFANA (SYSTEMD UNIT) ###############
sudo systemctl daemon-reload
sudo systemctl enable grafana-server.service
sudo systemctl start grafana-server