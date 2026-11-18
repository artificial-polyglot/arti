#!/bin/bash
set -e
set -x

if [ $# -ne 1 ]; then
    echo "usage: $0 version (e.g. v0.0.3)" >&2
    exit 1
fi
cd $GOPROJ
version="$1"
docker builder prune
docker build --platform linux/amd64 -f Docker/runpod_arti/Dockerfile -t runpod_arti .
docker login
docker tag runpod_arti garyngriswold/runpod_arti:${version}
docker push garyngriswold/runpod_arti:${version}
# Point the template and the serverless endpoint at the new image via the REST API v2 (v1 was
# retired 2026-11-15).  In v2 an endpoint keeps a copy of its template, so template edits no longer
# reach the endpoint; the endpoint must be updated itself.  Tracing is paused so the API key is not echoed.
set +x
image="{\"image\": \"garyngriswold/runpod_arti:${version}\"}"
curl --fail-with-body -sS -X PATCH "https://api.runpod.io/v2/templates/42n2voxks5" \
    -H "Authorization: Bearer ${RUNNING_PHESANT}" \
    -H "Content-Type: application/json" \
    -d "$image"
echo
curl --fail-with-body -sS -X PATCH "https://api.runpod.io/v2/serverless/fzfyyzux8a5dhi" \
    -H "Authorization: Bearer ${RUNNING_PHESANT}" \
    -H "Content-Type: application/json" \
    -d "$image"
echo
set -x
curl -d "build ${version} finished" https://ntfy.sh/arti2 \
    -H "Authorization: Bearer ${NTFY_API_TOKEN}"
sleep 60
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N1SKNSEC.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2ATGMLT.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2CCPBBS.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2MGUPNG.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2QAEBSP.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2SHNOMF.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2TTSXXX.yaml PROD
#python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/N2XNRPMS.yaml PROD
python Docker/runpod_arti/run_request.py /app/runpod_arti $HOME/arti2/P2LBEBTI.yaml PROD
curl -d "run pod ${version} finished" https://ntfy.sh/arti2 \
    -H "Authorization: Bearer ${NTFY_API_TOKEN}"

