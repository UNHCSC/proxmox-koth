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

# Alice is just a user. She should always be able to log in with her weak password.
# A player should make sure that Alice does not have any special permissions.
useradd -m -s /bin/bash alice
echo "alice:12345678" | chpasswd
usermod -aG sudo alice

# Remove alice from sudo group while preserving her user
gpasswd -d alice sudo

# Bob is a user who should have passwordless sudo access.
# A player should disable Bob's password login and ensure Bob uses his SSH key to log in.
useradd -m -s /bin/bash bob
echo "bob:5432123" | chpasswd
usermod -aG sudo bob
echo "bob ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/bob
mkdir /home/bob/.ssh
echo $2 > /home/bob/.ssh/authorized_keys

# TeamPoison is an evil hacker user. Their account should be locked, not deleted.
# A player should ensure that TeamPoison cannot log in.
useradd -m -s /bin/bash teampoison
echo "teampoison:teampoison" | chpasswd
usermod -aG sudo teampoison

## SGID allows us to create vulnerable directories.
# A player should ensure that these directories are not vulnerable by removing the SGID bit.
mkdir /var/toolkit
chown root:root /var/toolkit
chmod 2777 /var/toolkit

# Set up an evil script that can be run to take down services
echo "sudo service nginx stop" > /var/toolkit/evil.sh
chmod +x /var/toolkit/evil.sh

echo "Setup complete!"