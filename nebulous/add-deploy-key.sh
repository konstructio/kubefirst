#!/usr/bin/env bash

# ./scripts/nebulous/add-deploy-key.sh "YOUR_TOKEN" "YOUR_HOSTED_ZONE" "26"
# "${GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN}" "${HOSTED_ZONE_NAME}" "${GITLAB_KUBEFIRST_GROUP_ID}"

GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN=$1
GITLAB_URL=$2
GITLAB_KUBEFIRST_GROUP_ID=$3

export METAPHOR_PROJECT_ID=$(curl -s -XGET --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" "$GITLAB_URL/api/v4/projects?search=metaphor" | jq -r '.[].id')

# get filter with select working
DEPLOY_TOKEN_NAME=$(curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" "$GITLAB_URL/api/v4/projects/$METAPHOR_PROJECT_ID/deploy_keys" | jq -r '.[].title')

if [[ "$DEPLOY_TOKEN_NAME" == "gitlab-bot" ]]; then
  echo "deploy token named gitlab-bot already exists in metaphor"
else
  echo "deploy token named gitlab-bot is missing, creating it now"
  export GITLAB_BOT_SSH_PUBLIC_KEY_SRC=$(curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" "$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables/GITLAB_BOT_SSH_PUBLIC_KEY" | jq -r .value)
  echo $GITLAB_BOT_SSH_PUBLIC_KEY_SRC > tmp_pub_key.pub

  curl -s --request POST --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" --header "Content-Type: application/json" --data "{\"title\": \"gitlab-bot\", \"key\": \"$(cat ./tmp_pub_key.pub)\", \"can_push\": \"true\"}" "$GITLAB_URL/api/v4/projects/$METAPHOR_PROJECT_ID/deploy_keys/" > /dev/null
fi
