#!/usr/bin/env bash

set -e

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm

# aws profile setup
mkdir -p ~/.aws
cat << EOF > ~/.aws/config
[default]
output = json
region = ${AWS_DEFAULT_REGION}
EOF

cat << EOF > ~/.aws/credentials
[default]
aws_access_key_id = ${AWS_ACCESS_KEY_ID}
aws_secret_access_key = ${AWS_SECRET_ACCESS_KEY}
EOF
