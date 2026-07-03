# ---- Stage 1: build the Go binary ----
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/hello_arti ./cmd/dev/hello_arti

# ---- Stage 2: runtime with Python for the RunPod handler ----
FROM python:3.11-slim
WORKDIR /app
RUN pip install --no-cache-dir runpod
COPY --from=build /app/hello_arti /app/hello_arti
COPY cmd/dev/hello_arti/handler.py /app/handler.py
COPY cmd/dev/hello_arti/env.dev /app/env.dev
CMD ["/bin/sh", "-c", ". /app/env.dev && exec python -u /app/handler.py"]