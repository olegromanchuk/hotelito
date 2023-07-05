#!/bin/bash

# Initialize the path prefix with a default value
path_prefix="/hotelito-app"

# read the .env file
while read line; do
    # skip empty lines
    if [ -n "$line" ]; then
        # split the line into name and value
        name="${line%=*}"
        value="${line#*=}"

        # If this is the APPLICATION_NAME variable, update the path prefix
        if [ "$name" == "APPLICATION_NAME" ]; then
            path_prefix="/${value}"
        fi

        # create or update the parameter in the Parameter Store
        aws ssm put-parameter \
            --name "${path_prefix}/${name}" \
            --type "SecureString" \
            --value "${value}" \
            --overwrite
    fi
done < .env