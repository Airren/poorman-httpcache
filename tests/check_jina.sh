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


curl --location "http://127.0.0.1:${PORT}/jina/https://www.example.com" \
--header "Authorization: Bearer $JINA_API_KEY"

curl --location "https://r.jina.ai/https://www.example.com" \
--header "Authorization: Bearer $JINA_API_KEY"

