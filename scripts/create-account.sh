#!/usr/bin/env bash
# Usage: ./scripts/create-account.sh <username> <email> <password>
#
# Creates a Triage user account via the API.
# Reads TRIAGE_API_URL from the environment (defaults to https://triage.ckonkol.net).
#
# Example — create the MCP service account:
#   TRIAGE_API_URL=https://triage.ckonkol.net \
#     ./scripts/create-account.sh mcp mcp@local "$(openssl rand -hex 16)"

set -euo pipefail

TRIAGE_API_URL="${TRIAGE_API_URL:-https://triage.ckonkol.net}"

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <username> <email> <password>" >&2
  exit 1
fi

USERNAME="$1"
EMAIL="$2"
PASSWORD="$3"

echo "Creating account '$USERNAME' at $TRIAGE_API_URL ..."

RESPONSE=$(curl -sf \
  -X POST "$TRIAGE_API_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "$RESPONSE"

# If jq is available, pretty-print and extract the token
if command -v jq &>/dev/null; then
  echo ""
  echo "User ID : $(echo "$RESPONSE" | jq -r '.userId')"
  echo "Role    : $(echo "$RESPONSE" | jq -r '.role')"
  echo ""
  echo "Add these to your .env on the server:"
  echo "  MCP_USERNAME=$USERNAME"
  echo "  MCP_PASSWORD=$PASSWORD"
fi
