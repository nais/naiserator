# syntax=docker/dockerfile:1

FROM golang:1.27 AS builder
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.* /src/
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . /src
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/ ./cmd/naiserator ./cmd/naiserator_webhook

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app
COPY --from=builder /out/naiserator /app/naiserator
COPY --from=builder /out/naiserator_webhook /app/naiserator_webhook
CMD ["/app/naiserator"]
