#!/bin/bash

if [ "${AWS_PROFILE}" != "" ]; then
  echo "AWS_PROFILE is set (${AWS_PROFILE}), exporting session credentials"

  if aws sts get-caller-identity --profile "${AWS_PROFILE}" > /dev/null 2>&1; then
    echo "Using existing SSO credentials for profile ${AWS_PROFILE}"

    eval "$(aws configure export-credentials --profile ${AWS_PROFILE} --format env)"
  else
    if aws sso login --profile ${AWS_PROFILE}; then
      echo "SSO login successful."

      # Verify that credentials are available
      echo "Validating credentials..."
      if aws sts get-caller-identity --profile ${AWS_PROFILE} > /dev/null 2>&1; then
        echo "AWS session credentials are valid and active for profile: ${AWS_PROFILE}"
        eval "$(aws configure export-credentials --profile ${AWS_PROFILE} --format env)"
      else
        echo "AWS session credentials are not valid for profile ${AWS_PROFILE}"
        exit 1
      fi
    fi
  fi
fi

if [ -z "${AWS_ACCESS_KEY_ID}" ]; then
  if [ ! -f ~/.aws/credentials ]; then
    echo "ERROR: Missing AWS credentials!"
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

# Test that the credentials are valid
if ! aws sts get-caller-identity > /dev/null 2>&1; then
  echo "ERROR: Invalid AWS credentials!"
fi

docker-compose build metadata
DEFAULT_IAM_ROLE=$1 docker-compose run --rm metadata
