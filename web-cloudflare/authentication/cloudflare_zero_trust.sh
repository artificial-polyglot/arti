#!/usr/bin/env bash
# create-access-app.sh — protect the Arti bucket browser with Cloudflare Access
set -euo pipefail

: "${CLOUDFLARE_API_TOKEN:?set your API token (needs Access: Apps and Policies Write)}"
ACCOUNT_ID="your-account-id"      # <-- fill in
APP_DOMAIN="arti.example.com"     # <-- your Pages/Workers hostname

curl "https://api.cloudflare.com/client/v4/accounts/$ACCOUNT_ID/access/apps" \
 --request POST \
 --header "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
 --json '{
   "name": "Arti QA browser",
   "type": "self_hosted",
   "domain": "'"$APP_DOMAIN"'",
   "session_duration": "24h",
   "policies": [
     {
       "name": "Arti reviewers",
       "decision": "allow",
       "include": [
         { "email": { "email": "gary@shortsands.com" } }
       ]
     }
   ]
 }'
