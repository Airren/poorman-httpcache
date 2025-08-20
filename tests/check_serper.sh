#!/bin/bash

# Check if the search endpoint is working
echo "Checking search endpoint..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/../.env"
# export the env variable
set -o allexport
# shellcheck source=.env
source "$ENV_FILE"
set +o allexport

curl --location 'http://127.0.0.1:8080/serper/search' \
--header "X-API-KEY: $SERPER_API_KEY" \
--header 'Content-Type: application/json' \
--data '{"q":"apple inc"}'