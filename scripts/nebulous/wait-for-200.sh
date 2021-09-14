#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/wait-for-200.sh "https://gitlab-kubefirst.example.com"
###

set -e

URL=$1

while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' $URL)" != "200" ]]; 
do 
  echo "${URL} is not yet ready"
  sleep 10; 
done