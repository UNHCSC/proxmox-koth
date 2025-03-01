#!/bin/bash
echo "Setting up for team $1"

# Update and install packages
apt-get update && apt-get upgrade -y && apt-get install -y nginx python3 python3-pip

# Create user
useradd -m -s /bin/bash koth
echo "koth:password123" | chpasswd
usermod -aG sudo koth

# Create content to serve
echo "This is the main web page. Please do not change this" > /var/www/html/index.html
echo $1 > /var/www/html/team

# Enable and start Nginx
systemctl enable --now nginx

echo "Setup complete!"