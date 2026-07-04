# ---- Stage 1: build the Go binary ----
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/runpod_arti ./controller/runpod_arti

# ---- Stage 2: runtime with Python for the RunPod handler ----
FROM python:3.11-slim
ARG BASE=/workspace
# Data
ENV FCBH_DATASET_DB=${BASE}/data
ENV FCBH_DATASET_FILES=${BASE}/data/download
ENV FCBH_DATASET_TMP=${BASE}/data/tmp
# Conda ENVS
ENV FCBH_AENEAS_PYTHON=${BASE}/miniforge3/envs/aeneas/bin/python
ENV FCBH_FASTTEXT_EXE=${BASE}/miniforge3/envs/fasttext/fastText/fasttext
ENV FCBH_LIBROSA_PYTHON=${BASE}/miniforge3/envs/librosa/bin/python
ENV FCBH_MMS_ASR_PYTHON=${BASE}/miniforge3/envs/mms_asr/bin/python
ENV FCBH_MMS_FA_PYTHON=${BASE}/miniforge3/envs/mms_fa/bin/python
ENV FCBH_MMS_ADAPTER_PYTHON=${BASE}/miniforge3/envs/mms_adapter/bin/python
ENV FCBH_UROMAN_EXE=${BASE}/miniforge3/envs/mms_fa/bin/uroman.pl
ENV FCBH_WHISPER_EXE=${BASE}/miniforge3/envs/whisper/bin/whisper
# Logging
ENV FCBH_DATASET_LOG_FILE=${BASE}/arti.log
ENV FCBH_DATASET_LOG_LEVEL=DEBUG
# Cuda
ENV PYTORCH_CUDA_ALLOC_CONF=expandable_segments:True
# Buckets
ENV FCBH_DATASET_IN_BUCKET=arti-input
ENV FCBH_DATASET_IO_BUCKET=arti-output
RUN mkdir -p "$FCBH_DATASET_DB" "$FCBH_DATASET_FILES" "$FCBH_DATASET_TMP"

WORKDIR /app
RUN pip install --no-cache-dir runpod
COPY --from=build /app/runpod_arti /app/runpod_arti
COPY controller/runpod_arti/handler.py /app/handler.py
COPY controller/runpod_arti/env.secret /app/env.secret
CMD ["python", "-u", "/app/handler.py"]