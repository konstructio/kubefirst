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


# to unpin, remove the version from the end of the command as shown here
# sudo EXTERNAL_URL="https://${GITLAB_URL}" GITLAB_ROOT_PASSWORD="${GITLAB_BOT_ROOT_PASSWORD}" apt-get install gitlab-ce
# for list of releases see: https://about.gitlab.com/releases/categories/releases/
sudo EXTERNAL_URL="https://${GITLAB_URL}" GITLAB_ROOT_PASSWORD="${GITLAB_BOT_ROOT_PASSWORD}" apt-get install gitlab-ce=14.9.2-ce.0
