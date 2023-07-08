#!/bin/bash

#This script will read the .env file and create/update the parameters in the AWS Parameter Store and also create environment variables in current shell

# Initialize the path prefix with a default value
path_prefix="/hotelito-app"

## check .env in local dir and in the root dir. Local dir takes precedence
if [ -f ../../.env ]; then
  ENV_FILE=../../.env
fi

if [ -f .env ]; then
  ENV_FILE=.env
fi

if [[ -z "$ENV_FILE" ]]; then
  echo "Create .env file first. Check .env_example for required parameters"
  exit 1
fi

echo "Using file: ${ENV_FILE}"

# read the .env file
while read line; do
  if [ -n "$line" ]; then
    # split the line into name and value
    name="${line%=*}"
    value="${line#*=}"

    # If this is the APPLICATION_NAME variable, update the path prefix
    if [ "$name" == "APPLICATION_NAME" ]; then
      path_prefix="/${value}"
      export APP_NAME="${value}"
    fi
    if [ "$name" == "ENVIRONMENT" ]; then
      path_prefix_env="/${value}"
      export ENVIRONMENT="${value}"
    fi
    if [ "$name" == "LOG_LEVEL" ]; then
      export LOG_LEVEL="${value}"
    fi
  fi
done <${ENV_FILE}

# read the .env file
while read line; do
  # skip empty lines
  if [ -n "$line" ]; then
    # split the line into name and value
    name="${line%=*}"
    value="${line#*=}"

    # create or update the parameter in the Parameter Store
    echo "Setting ${path_prefix}/${path_prefix_env}/${name}"
    aws ssm put-parameter \
      --name "${path_prefix}/${path_prefix_env}/${name}" \
      --type "SecureString" \
      --value "${value}" \
      --overwrite
  fi
done <${ENV_FILE}

#1. create
BUCKET_ID=$(dd if=/dev/random bs=8 count=1 2>/dev/null | od -An -tx1 | tr -d ' \t\n')
BUCKET_NAME=lambda-artifacts-$BUCKET_ID
echo $BUCKET_NAME >bucket-name.txt
aws s3 mb s3://$BUCKET_NAME --profile=${AWS_CONFIG_PROFILE}
aws cloudformation package --template-file template-deploy-via-sam.yml --s3-bucket $ARTIFACT_BUCKET --output-template-file template-deploy-via-sam-export.yml --profile=${AWS_CONFIG_PROFILE}