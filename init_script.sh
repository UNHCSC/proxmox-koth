#!/bin/bash

# Update and install packages
apt-get update && apt-get upgrade -y && apt-get install -y nginx python3 python3-pip

# Install node, refresh sources, install pm2 for process management
wget -qO- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
nvm install --lts
npm i -g pm2
pm2 startup

# Create user
useradd -m -s /bin/bash csc
echo "csc:password123" | chpasswd

# Create content to serve
echo "If you can see this, your node is online" > /var/www/html/index.html

# Enable and start Nginx
systemctl enable --now nginx

echo "Setup complete!"