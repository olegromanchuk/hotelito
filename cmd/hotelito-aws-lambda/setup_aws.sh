#!/bin/bash

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