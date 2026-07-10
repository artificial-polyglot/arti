#!/bin/bash
set -e
set -x

if [ $# -ne 1 ]; then
    echo "usage: $0 version (e.g. v0.0.3)" >&2
    exit 1
fi
cd $GOPROJ
version="$1"
docker build --platform linux/amd64 -f controller/runpod_arti/Dockerfile -t runpod_arti .
docker login
docker tag runpod_arti garyngriswold/runpod_arti:${version}
docker push garyngriswold/runpod_arti:${version}
runpodctl template update "42n2voxks5" --image "garyngriswold/runpod_arti:${version}"
curl -d "build ${version} finished" https://ntfy.sh/artificial-polyglot
sleep 15
python controller/runpod_arti/run_request.py $HOME/arti2/N2QAEBSP_trainonly.yaml PROD

