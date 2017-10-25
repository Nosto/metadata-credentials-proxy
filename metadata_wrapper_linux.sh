#!/bin/sh

if [ -z "${AWS_ACCESS_KEY_ID}" ]; then
  echo "Missing AWS_ACCESS_KEY_ID env variable!"
  exit 1
fi

if [ -z "${AWS_SECRET_KEY}" ]; then
  echo "Missing AWS_SECRET_KEY env variable!"
  exit 1
fi

echo "Starting metadata server..."
if [ -n "${DEFAULT_IAM_ROLE}" ]; then
  echo "DEFAULT_IAM_ROLE=${DEFAULT_IAM_ROLE}"
else
  echo "DEFAULT_IAM_ROLE not set"
fi

./metadata
