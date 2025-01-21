#!/bin/bash

if [ -z "${AWS_ACCESS_KEY_ID}" ]; then
  if [ ! -f ~/.aws/credentials ]; then
    echo "ERROR: ~/.aws/credentials does not exist and AWS_ACCESS_KEY_ID and AWS_SECRET_KEY are not set!"
    exit 1
  fi

  aws_access_key_id=$(sed -n 's/.*aws_access_key_id *= *\([^ ]*.*\)/\1/p' < ~/.aws/credentials)
  aws_secret_access_key=$(sed -n 's/.*aws_secret_access_key *= *\([^ ]*.*\)/\1/p' < ~/.aws/credentials)

  if [ "${aws_access_key_id}" == "" ]; then
    echo "Cannot read access key id from ~/.aws/credentials"
    exit 1
  fi

  if [ "${aws_secret_access_key}" == "" ]; then
    echo "Cannot read access key id from ~/.aws/credentials"
    exit 1
  fi

  export AWS_ACCESS_KEY_ID=$aws_access_key_id
  export AWS_SECRET_KEY=$aws_secret_access_key
  export AWS_SECRET_ACCESS_KEY=$aws_secret_access_key

  echo "Using AWS credentials from ~/.aws/credentials"
fi

if ! docker network ls --format "{{.Name}}" | grep -q -x metadata; then
  echo "Adding missing metadata network"
  docker network create -d bridge --subnet 169.254.169.0/24 metadata
fi

docker compose build metadata
DEFAULT_IAM_ROLE=$1 docker compose run --rm metadata
