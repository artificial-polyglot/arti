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

# Install curl (needed to fetch the Miniforge installer)
RUN apt-get update && apt-get install -y --no-install-recommends curl && \
    rm -rf /var/lib/apt/lists/*

# Install Miniforge
RUN curl -L -o /tmp/miniforge.sh \
    https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-x86_64.sh && \
    bash /tmp/miniforge.sh -b -p ${BASE}/miniforge3 && \
    rm /tmp/miniforge.sh

# Install runpod
RUN ${BASE}/miniforge3/bin/pip install --no-cache-dir runpod

# Install base executables into the base environment (no activation needed)
RUN ${BASE}/miniforge3/bin/conda install -y -n base -c conda-forge ffmpeg sox sqlite && \
    ${BASE}/miniforge3/bin/conda clean -afy
ENV PATH=${BASE}/miniforge3/bin/:$PATH

# Create and populate the mms_fa environment
ARG MMS_FA=${BASE}/miniforge3/envs/mms_fa/bin

RUN ${BASE}/miniforge3/bin/conda create -y -n mms_fa python=3.11

RUN ${MMS_FA}/pip install --no-cache-dir --retries 5 --timeout 120 torch torchaudio \
    --index-url https://download.pytorch.org/whl/cu128

RUN ${MMS_FA}/pip install --no-cache-dir --retries 5 --timeout 120 torchcodec \
    --index-url https://download.pytorch.org/whl/cu128

RUN ${BASE}/miniforge3/bin/conda install -y -n mms_fa -c conda-forge pysoundfile ffmpeg-python sox && \
    ${MMS_FA}/pip install sox && \
    ${MMS_FA}/pip install uroman && \
    cp ${MMS_FA}/uroman ${MMS_FA}/uroman.pl && \
    ${MMS_FA}/pip cache purge && \
    ${BASE}/miniforge3/bin/conda clean -afy
RUN ${MMS_FA}/pip install --no-cache-dir nvidia-npp-cu12
ENV LD_LIBRARY_PATH=${BASE}/miniforge3/envs/mms_fa/lib/python3.11/site-packages/nvidia/npp/lib:$LD_LIBRARY_PATH

# Pre-download the MMS_FA model weights so the container never fetches them
# over the network at runtime.
ENV TORCH_HOME=${BASE}/torch_cache
RUN ${MMS_FA}/python -c "from torchaudio.pipelines import MMS_FA as bundle; bundle.get_model(with_star=False)"

# Copy over python code
ENV GOPROJ=/app/arti
COPY --from=build /src ${GOPROJ}
RUN find ${GOPROJ} -name "*.go" -delete

COPY --from=build /app/runpod_arti /app/runpod_arti
COPY controller/runpod_arti/handler.py /app/handler.py
CMD ["python", "-u", "/app/handler.py"]