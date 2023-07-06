#!/bin/bash


# Specify output JSON file
JSON_FILE="environmental_vars.json"

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

# Remove JSON file if it already exists
if [ -f $JSON_FILE ]; then
    rm $JSON_FILE
fi

# Add opening brace to start JSON
echo "{
  \"HotelitoFunction\": {" >> $JSON_FILE

while IFS='=' read -r key value
do
    if [ -z "$key" ] || [[ $key == \#* ]]; then
        # Skip empty lines and lines starting with #
        continue
    fi

    # Remove quotes if they exist
    value=${value%\"}
    value=${value#\"}
    # Escape special characters
    value=${value//\\/\\\\}
    value=${value//\"/\\\"}
    # Write to JSON file
    echo "    \"$key\": \"$value\"," >> $JSON_FILE
done < "$ENV_FILE"

# Remove last comma
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX
    sed -i '' -e '$ s/,$//' "$JSON_FILE"
else
    # Linux
    sed -i '$ s/,$//' "$JSON_FILE"
fi

# Add closing brace to end JSON
echo "  }
}" >> $JSON_FILE
