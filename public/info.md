# King of the Hill 2025

Sponsored by the [University of New Hampshire Cybersecurity Club](https://cyber.cs.unh.edu/)

## Overview

This is a King of the Hill competition where teams will compete to defend a Linux server, and attack other team's servers. The server will be a Ubuntu 24.04 LXC container, which is a lightweight virtual machine. The server will be running a few services, which teams are tasked with securing and maintaining. Teams are also allowed to red team (attack) other team's servers, and can gain points for doing so. The team with the most points at the end of the competition wins.

## Technologies Involved

- Linux (Ubuntu 24.04 LXC container)
    - Be familiar with:
        - Bash (shell scripting)
        - File permissions
        - Package management (apt)
        - Users and groups
        - Networking (ifconfig, netstat, etc.)
- SSH (OpenSSH, OpenSSL)
    - Be familiar with different authentication methods (password, key, etc.)
- Web Server (Nginx)
    - Be familiar with web server configuration and security
    - Understand where files are stored (/var/www/html/)
- Firewall (UFW)
    - Be familiar with firewall configuration and security
    - Understand how to allow/deny traffic to/from specific ports
    - Avoid locking yourself out of your own server on SSH
- Database (MySQL)
    - Be familiar with database configuration and security
    - Understand how to create databases, users, and tables
    - Avoid SQL injection attacks
- REST API (Flask) **(Will be used to interact with the database)**
    - Be familiar with REST API configuration and security
    - Understand how to create routes and handle requests
    - Avoid SQL injection attacks

## Scoring Breakdown

- Minutely checks
- Some checks require SSH access to your container
    - This will **almost always** be done via the black team

### Defense

- `+1` point if the container is able to be pinged on IPv4 (ICMP)
- `+2` points if "black-team" user can log in with SSH public key authentication
- `+3` points if the web server is running and serving a web page
- `+2` points if the file at `/var/www/html/team` has the content `Team X`, where `X` is your team's number
- `+2` points if the database is online and the "black-team" user can access it
- `+2` points if the REST API is online, and the "black-team" user can GET `/api/my-secret` using credentials stored in `/home/black-team/.flaskenv`
- `+4` points if the "black-team" user can POST `/api/my-secret` and update it, and ensure it was updated by GET `/api/my-secret`
    > This will test that the REST API is allowing reads and writes, which requires teams to hunt down credentials rather than just stopping POST requests

### Offense

- `+5` points if your team's number is found in the `/var/www/html/team` file on another team's container
    - This is stackable, so if you find multiple teams' numbers, you get points for each
- `+5` points if you can overwrite the `OWNED_BY` record in the database with your team's number
    - This is stackable, so if you find multiple teams' numbers, you get points for each

## Rules

- "black-team" user is **out of play**
    - Do not disable, change authentication, or delete this user
    - Do not log in using this user
    - Do not attempt to exploit this user
- Do not exploit zero-day vulnerabilities
- Do not attack the competition infrastructure (or anything outside of the specific given IPv4 CIDR block)

## Good Faith and Spirit of the Game

- This is a competition, but it is also a learning experience
- Please avoid `rm -rf /` or other unrecoverable attacks
- Please avoid locking other teams out of their containers too early on
- Avoid blocking IP ranges of other users, and instead focus on securing your own container
    - If you were part of a company and blocked everyone from seeing your website, what would your boss think?
- You can ask for help if you're stuck, but attempt to solve the problem yourself first

## Teams and Staff

### Blue Team

Every team of competitors is a "Blue Team". They are tasked with defending their container and attacking other teams' containers.

### Red Team

The "Red Team" is the group of students and/or staff who want to cause some chaos. They will slowly be attacking containers, and will start with the same information as everybody else. They will not be allowed to nuke containers until the very end, but will be encouraged to cause some easily recoverable chaos. Most of the attacks will be automated.

### Black Team

The "Black Team" is the group of students and/or staff who are responsible for the competition infrastructure. They will be responsible for setting up the containers, monitoring the containers, and ensuring that the competition runs smoothly. They will also be responsible for scoring the competition.

## Helpful Tips from Evan

1. Backup your database, firewall, /var/www/html, and other important files regularly
2. Have a plan for what to do if you get locked out of your container
3. Don't lock yourself out of your box with `ufw`, make sure you allow SSH!
    - (If this does happen and we can confirm it's not a malicious attack, we can help you out)
4. Don't forget to check your logs for suspicious activity
5. `pkill -u username -KILL` will kick off a user from the container
6. `who` will show you who is logged in
7. You can hide bash scripts in `.bashrc` to do things when someone's session starts and they enter the bash shell
