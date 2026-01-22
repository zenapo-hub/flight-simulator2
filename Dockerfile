# syntax=docker/dockerfile:1.4

FROM --platform=$BUILDPLATFORM golang:1.23.0-alpine3.19 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o server ./cmd/server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080
ENTRYPOINT ["/app/server"]
