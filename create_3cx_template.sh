#!/bin/bash

ORIGINAL_FILE_3CX=3cx/src/crm-template-cloudbeds-3cx-template.xml
FINAL_FILE_3CX=3cx/crm-template-cloudbeds-3cx.xml
ENV_FILE=.env

# read the .env file
while read line; do
  if [ -n "$line" ]; then
    # split the line into name and value
    name="${line%=*}"
    value="${line#*=}"

    # If this is the APPLICATION_NAME variable, update the path prefix
    if [ "$name" == "CLOUDBEDS_REDIRECT_URL" ]; then
      # CLOUDBEDS_REDIRECT_URL=https://8571-22.43.2.5.555.ngrok-free.app/api/v1/callback
      API_BASE_URL=$(echo ${value} | sed -e 's|\/api\/v1\/callback||g')
    fi
  fi
done <${ENV_FILE}

# create copy of template
cp -prf ${ORIGINAL_FILE_3CX} ${FINAL_FILE_3CX}

# replace the TEMPLATE_API_URL on API_BASE_URL
sed -i "" -e "s|TEMPLATE_API_URL|${API_BASE_URL}|" ${FINAL_FILE_3CX}
echo "Updated ${FINAL_FILE_3CX} with API_BASE_URL=${API_BASE_URL}"
echo "Import it in 3CX as a new template."