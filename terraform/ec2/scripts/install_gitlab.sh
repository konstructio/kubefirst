#!/usr/bin/env bash
sudo apt-get update
sudo apt-get install -y curl openssh-server ca-certificates
​
sudo debconf-set-selections <<< "postfix postfix/mailname string ${EMAIL_DOMAIN}"
sudo debconf-set-selections <<< "postfix postfix/main_mailer_type string 'Internet Site'"
​
sudo apt-get install --assume-yes postfix
​
curl https://packages.gitlab.com/install/repositories/gitlab/gitlab-ce/script.deb.sh | sudo bash

sudo EXTERNAL_URL="https://${GITLAB_URL}" apt-get install gitlab-ce=13.12.9-ce.0
