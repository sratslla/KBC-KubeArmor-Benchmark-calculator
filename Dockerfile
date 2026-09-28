# Multi-stage build for in-cluster Job runs.
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /kbc ./cmd/kbc

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
  && curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash \
  && rm -rf /var/lib/apt/lists/*
COPY --from=build /kbc /usr/local/bin/kbc
COPY deploy/manifests /configs/deploy/manifests
COPY configs /configs/configs
WORKDIR /work
ENTRYPOINT ["kbc"]
