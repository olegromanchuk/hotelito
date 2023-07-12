#!/bin/bash
set -o errexit
#This script will read the .env file and create/update the parameters in the AWS Parameter Store and also create environment variables in current shell

# get the AWS profile from the samconfig.toml file
AWS_CONFIG_PROFILE=$(grep 'profile' samconfig.toml | awk -F '"' '{print $2}' | cut -d'"' -f 1)

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
      path_prefix_env="${value}"
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
    aws --profile ${AWS_CONFIG_PROFILE} ssm put-parameter \
      --name "${path_prefix}/${path_prefix_env}/${name}" \
      --type "SecureString" \
      --value "\"${value}\"" \
      --overwrite
  fi
done <${ENV_FILE}


# 2. Prepare for deployment
AWS_CONFIG_PROFILE=$(grep 'profile' samconfig.toml | awk -F '"' '{print $2}' | cut -d'"' -f 1)
STACKNAME=$(grep 'stack_name' samconfig.toml | awk -F '"' '{print $2}' | cut -d'"' -f 1)

if [[ -z "$AWS_CONFIG_PROFILE" ]]; then
  echo "AWS_CONFIG_PROFILE is not set. Run \"sam deploy --guided\" first"
  exit 1
fi

if [[ -z "$STACKNAME" ]]; then
  echo "STACKNAME is not set. Run \"sam deploy --guided\" first"
  exit 1
fi

echo "Using AWS_CONFIG_PROFILE: ${AWS_CONFIG_PROFILE}"
echo "Using STACKNAME: ${STACKNAME}"
echo "Using APP_NAME: ${APP_NAME}"
echo "Using ENVIRONMENT: ${ENVIRONMENT}"

# 2. Deploy via SAM
sam deploy --profile ${AWS_CONFIG_PROFILE}

# 3. Update the API Gateway throttling settings
read -p "set throttling? yes/no: " DEVEL
if [[ ${DEVEL} == "yes" ]]; then
  APIID=$(aws apigateway get-rest-apis --profile=${AWS_CONFIG_PROFILE} | jq -r ".items[] | select (.name == \"${STACKNAME}\") | .id")
  echo "APIID: ${APIID}, AWS_CONFIG_PROFILE: ${AWS_CONFIG_PROFILE}, STACKNAME: ${STACKNAME}"
  aws apigateway delete-stage --rest-api-id="${APIID}" --stage-name="Stage" --profile=${AWS_CONFIG_PROFILE}
  aws apigateway update-stage --rest-api-id="${APIID}" --stage-name="Prod" --patch-operations op=replace,path='/*/*/throttling/rateLimit',value=4 --profile=${AWS_CONFIG_PROFILE}
  aws apigateway update-stage --rest-api-id="${APIID}" --stage-name="Prod" --patch-operations op=replace,path='/*/*/throttling/burstLimit',value=2 --profile=${AWS_CONFIG_PROFILE}
fi

# 4. Get API gateway URL
aws apigateway --profile demo-3cx-account get-rest-apis --query "items[?name=='hotelito-go1'].id"