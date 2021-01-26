#!/usr/bin/env bash

set -e

GITLAB_URL=$1

while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' $GITLAB_URL)" != "200" ]]; 
do 
  echo "${GITLAB_URL} is not yet ready, sleeping for 2 min"
  sleep 120; 
done