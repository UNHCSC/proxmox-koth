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

# Create content to serve
echo "This is the main web page. Please do not change this" > /var/www/html/index.html
echo $1 > /var/www/html/team

# Enable and start Nginx
systemctl enable --now nginx

newUsers=("Anthony Phelps" "Sebastian Vaughn" "Colby Fisher" "Bronte Roberson" "Gordon Thornton" "Aoife Rocha" "Nikolas Klein" "Evan Kasper" "John UNH")

for user in "${newUsers[@]}"; do
    username=$(echo "$user" | tr '[:upper:]' '[:lower:]' | tr ' ' '.')
    if [[ ! " ${USERS[*]} " =~ " ${username} " ]]; then
        USERS+=("$username")
        useradd -m -s /bin/bash "$username"
        chfn -f "$user" "$username"
        echo "$username:password" | chpasswd
        echo "$username ALL=(ALL) NOPASSWD: ALL" >>/etc/sudoers
    fi
done


echo "Setup complete!"