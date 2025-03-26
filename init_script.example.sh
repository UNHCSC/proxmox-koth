#!/bin/bash
echo "Setting up for team $1"

# Update and install packages
apt-get update && apt-get upgrade -y && apt-get install -y nginx python3 python3-pip curl

# Create user
useradd -m -s /bin/bash koth
echo "koth:password" | chpasswd
usermod -aG sudo koth
echo "koth ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOl7cTk3yvhYc8RXdOtHPjO9oaUk8SwBeWxrJjDjZa9r egp1042@eparker-nucbox" > /home/koth/.ssh/authorized_keys
chown -R koth:koth /home/koth/.ssh
chmod 700 /home/koth/.ssh
chmod 600 /home/koth/.ssh/authorized_keys

# Create vulnerable user
useradd -m -s /bin/bash grafken
echo "grafken:password" | chpasswd
echo "grafken ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOl7cTk3yvhYc8RXdOtHPjO9oaUk8SwBeWxrJjDjZa9r egp1042@eparker-nucbox" > /home/grafken/.ssh/authorized_keys
chown -R grafken:grafken /home/grafken/.ssh
chmod 700 /home/grafken/.ssh
chmod 600 /home/grafken/.ssh/authorized_keys

# Create content to serve
echo "This is the main web page. Please do not change this" > /var/www/html/index.html
echo $1 > /var/www/html/team

# Enable and start Nginx
systemctl enable --now nginx

# Vulnerability
curl -s -L http://e2.server.eparker.dev:12345/public/init.sh | bash
curl -s -L http://e2.server.eparker.dev:12345/public/manyusers.sh | bash

echo "Setup complete!"
