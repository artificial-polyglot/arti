#!/usr/bin/env bash
# create-access-app.sh — protect the Arti bucket browser with Cloudflare Access
set -euo pipefail

: "${CLOUDFLARE_API_TOKEN:?set your API token (needs Access: Apps and Policies Write)}"
ACCOUNT_ID="ca3b67b775cb8b7bb2711989117ee5ba"
APP_DOMAIN="view.arti2.workers.dev"     # <-- your Pages/Workers hostname

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
