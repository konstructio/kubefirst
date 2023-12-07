#!/bin/bash

#
# This script helps you assume the AssumedAdmin role you either created manually or using the Terraform plan from ws-create-role.tf
#
# Requirement: aws-cli installed (see https://github.com/aws/aws-cli)
#
# Replace the AWS account ID `111111111111` in the `ROLE` variable with yours. If you give the admin a different name than `AssumedAdmin`, please update it also.
#
# Ensure that the default values fit your needs (i.e., role session name, duration of assume role...)
#
# Before running the script, ensure you have credentials. configure with the AWS CLI. To do so, run
# aws configure
#
# To run this script
# ./aws-assume-role.sh
#

#
# Change the AWS account ID & role name
#
ROLE="arn:aws:iam::126827061464:role/KubernetesAdmin"

#
# You can leave the rest of thre script as is
#

# An identifier for the assumed role session: you can change it if you want.
ROLE_SESSION_NAME="AssumedAdmin-kubefirst"

# Colors for formatting
YELLOW="\033[1;93m"
NOFORMAT="\033[0m"
BOLD="\033[1m"

# Backup the previous credentials
if [ -f "~/.aws/credentials" ]
then
    mv ~/.aws/credentials ~/.aws/credentials.bak
fi

# Unset previously set AWS access environment variables
unset AWS_ACCESS_KEY_ID
unset AWS_SECRET_ACCESS_KEY
unset AWS_SESSION_TOKEN

# Retrieving the connected user
USER=$(aws sts get-caller-identity | jq -r .Arn | cut -d'/' -f 2)

if [ $(echo "$USER" | grep -v "Unable to locate credentials") ]
then
    # Assuming the new role for 12 hours. You can change the `--duration-seconds` to shorter timeout for security reason.
    JSON=$(aws sts assume-role --role-arn "${ROLE}" --role-session-name "${ROLE_SESSION_NAME}" --duration-seconds 43200)
    export AWS_ACCESS_KEY_ID=$(echo $JSON | jq -r .Credentials.AccessKeyId)
    export AWS_SECRET_ACCESS_KEY=$(echo $JSON | jq -r .Credentials.SecretAccessKey)
    export AWS_SESSION_TOKEN=$(echo $JSON | jq -r .Credentials.SessionToken)
    unset JSON

    # Display useful information for UI installation
    echo -e "\n${YELLOW}Started session for user ${NOFORMAT}${BOLD}${USER}${NOFORMAT}${YELLOW} assuming ${NOFORMAT}${BOLD}${ROLE}${NOFORMAT}\n"
    echo -e "${BOLD}AWS_ACCESS_KEY_ID:    ${NOFORMAT} ${AWS_ACCESS_KEY_ID}"
    echo -e "${BOLD}AWS_SECRET_ACCESS_KEY:${NOFORMAT} ${AWS_SECRET_ACCESS_KEY}"
    echo -e "${BOLD}AWS_SESSION_TOKEN:    ${NOFORMAT} ${AWS_SESSION_TOKEN}"
else
    # The script wasn't successful
    exit 1
fi
