#!/bin/bash
set -o errexit
#This script will read the .env file and create/update the parameters in the AWS Parameter Store and also create environment variables in current shell

ORIGINAL_FILE_3CX=../../3cx/src/crm-template-cloudbeds-3cx-template.xml
FINAL_FILE_3CX=../../3cx/crm-template-cloudbeds-3cx.xml
FILE_ROOMID_EXTENSION_MAP=../../roomid_map.json

# if samconfig.toml doesn't exist - run sam deploy --guided
if [ ! -f samconfig.toml ]; then
  echo "samconfig.toml doesn't exist. Running \"sam deploy --guided\" first"
  sam deploy --guided
fi

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
while IFS= read -r line || [[ -n "$line" ]]; do
  if [ -n "$line" ]; then
    # split the line into name and value
    name="${line%=*}"
    value="${line#*=}"

    # If this is the APPLICATION_NAME variable, update the path prefix
    if [ "$name" == "APPLICATION_NAME" ]; then
      path_prefix="/${value}"
      export APPLICATION_NAME="${value}"
    fi
    if [ "$name" == "ENVIRONMENT" ]; then
      path_prefix_env="${value}"
      export ENVIRONMENT="${value}"
    fi
    if [ "$name" == "LOG_LEVEL" ]; then
      export LOG_LEVEL="${value}"
    fi
    if [ "$name" == "AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID" ]; then
      export AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID="${value}"
    fi
  fi
done <${ENV_FILE}

# read the .env file
while IFS= read -r line || [[ -n "$line" ]]; do
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
echo "Using APPLICATION_NAME: ${APPLICATION_NAME}"
echo "Using ENVIRONMENT: ${ENVIRONMENT}"

# 2. Deploy via SAM
# TODO - get real Environment and ApplicationName fromn ./env and pass it to "sam deploy"
sam deploy --profile ${AWS_CONFIG_PROFILE} --no-confirm-changeset --parameter-overrides "ParameterKey=LogLevel,ParameterValue=${LOG_LEVEL} ParameterKey=S3BucketMap3CXRoomExtClBedsRoomId,ParameterValue=${AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID} ParameterKey=ApplicationName,ParameterValue=${APPLICATION_NAME} ParameterKey=Environment,ParameterValue=${ENVIRONMENT}"
#sam deploy --profile ${AWS_CONFIG_PROFILE} --parameter-overrides "ParameterKey=LogLevel,ParameterValue=${LOG_LEVEL}"


# 3. Update the API Gateway throttling settings
read -p "set throttling? yes/no: " DEVEL
if [[ ${DEVEL} == "yes" ]]; then
  APIID=$(aws --profile=${AWS_CONFIG_PROFILE} apigateway get-rest-apis | jq -r ".items[] | select (.name == \"${STACKNAME}\") | .id")
  echo "APIID: ${APIID}, AWS_CONFIG_PROFILE: ${AWS_CONFIG_PROFILE}, STACKNAME: ${STACKNAME}"
  aws --profile=${AWS_CONFIG_PROFILE} apigateway update-stage --rest-api-id="${APIID}" --stage-name="Prod" --patch-operations op=replace,path='/*/*/throttling/rateLimit',value=4
  aws --profile=${AWS_CONFIG_PROFILE} apigateway update-stage --rest-api-id="${APIID}" --stage-name="Prod" --patch-operations op=replace,path='/*/*/throttling/burstLimit',value=2

  #delete stage Stage if any

  # Get the stages
  STAGES=$(aws --profile=${AWS_CONFIG_PROFILE apigateway get-stages --rest-api-id="${APIID}"} --query 'item[*].stageName' --output text)

  # Check if the stage exists in the list
  if echo "$STAGES" | grep -q "Stage"; then
      echo "Stage exists. Deleting..."
      aws --profile=${AWS_CONFIG_PROFILE} apigateway delete-stage --rest-api-id="${APIID}" --stage-name="Stage"
  else
      echo "Stage does not exist."
  fi
fi

# 4. Get API gateway URL
FUNC_NAME=$(aws --profile ${AWS_CONFIG_PROFILE} apigateway get-rest-apis --query "items[?name=='${STACKNAME}'].id" --output text)
# get the AWS region from the samconfig.toml file
AWS_REGION=$(grep '^region' samconfig.toml | awk -F '"' '{print $2}' | cut -d'"' -f 1)
# compiling the URL. The URL will be used by cloudbeds to redirect the user after login back to our API
CLOUDBEDS_REDIRECT_URL="https://${FUNC_NAME}.execute-api.${AWS_REGION}.amazonaws.com/Prod/api/v1/callback"
# set ssm parameter
name="CLOUDBEDS_REDIRECT_URL"
value="${CLOUDBEDS_REDIRECT_URL}"
# create or update the parameter in the Parameter Store
echo "Setting ${path_prefix}/${path_prefix_env}/${name} in parameter store"
aws --profile ${AWS_CONFIG_PROFILE} ssm put-parameter \
  --name "${path_prefix}/${path_prefix_env}/${name}" \
  --type "SecureString" \
  --value "\"${value}\"" \
  --overwrite

# 5. create 3CX template
# create copy of template
cp -prf ${ORIGINAL_FILE_3CX} ${FINAL_FILE_3CX}
API_BASE_URL="https://${FUNC_NAME}.execute-api.${AWS_REGION}.amazonaws.com/Prod"

# replace the TEMPLATE_API_URL on API_BASE_URL
sed -i "" -e "s|TEMPLATE_API_URL|${API_BASE_URL}|" ${FINAL_FILE_3CX}


# 6. Upload roomid_map.json to S3
# Set the bucket name variable
echo "Uploading ${FILE_ROOMID_EXTENSION_MAP} to s3://${AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID}"
# Upload the file to S3 bucket
aws --profile ${AWS_CONFIG_PROFILE} s3 cp "${FILE_ROOMID_EXTENSION_MAP}" "s3://${AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID}/"


# 6. Output results
echo "--------------"
echo "Set \"REDIRECT URL\" in cloudbeds to this value: ${CLOUDBEDS_REDIRECT_URL}"
echo
echo "For initial authentication run: https://${FUNC_NAME}.execute-api.${AWS_REGION}.amazonaws.com/Prod/api/v1"
echo "3CX template could be found here: ${FINAL_FILE_3CX} Import it in 3CX as a new template."
echo
