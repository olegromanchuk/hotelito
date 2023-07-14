#!/bin/bash

# Define the SAM template filename
TEMPLATE_FILENAME="template.yaml"

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

add_section_to_json() {
  local section_name=$1
  echo "\"${section_name}\": {" >>$JSON_FILE

  while IFS='=' read -r key value || [[ -n "$key" ]]; do
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
    echo "    \"$key\": \"$value\"," >>$JSON_FILE
  done <"$ENV_FILE"

  # Remove last comma
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX
    sed -i '' -e '$ s/,$//' "$JSON_FILE"
  else
    # Linux
    sed -i '$ s/,$//' "$JSON_FILE"
  fi

  # Add closing brace to end JSON
  echo "  }," >>$JSON_FILE
  # End section
}

# Add opening brace to start JSON
echo "{" >>$JSON_FILE

##some magic here. Extract function names from template.yml and add them to the json file

# Read the template file and print function names followed by "Type: AWS::Serverless::Function"
# Search for next two lines
#  SomeFunction:
#    Type: AWS::Serverless::Function
# and extract "SomeFunction"
awk '/Type: AWS::Serverless::Function/ {gsub(/:$/, "", prev_line); gsub(/^ +/, "", prev_line); print prev_line} {prev_line=$0}' $TEMPLATE_FILENAME > functions.txt

# Declare an array to store function names
declare -a function_names

# Read the file line by line into an array
while IFS= read -r line; do
    add_section_to_json "$line"
done < functions.txt

# Clean up the temporary file
rm functions.txt


# Remove last comma
if [[ "$OSTYPE" == "darwin"* ]]; then
  # Mac OSX
  sed -i '' -e '$ s/,$//' "$JSON_FILE"
else
  # Linux
  sed -i '$ s/,$//' "$JSON_FILE"
fi

# Add final brace to end JSON
echo "}" >>$JSON_FILE
