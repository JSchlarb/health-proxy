FROM --platform=${BUILDPLATFORM} golang:1.24.5-alpine AS builder

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org
ENV GOCACHE=/go/.cache/go-build
ENV GOMODCACHE=/go/.cache/mod

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/.cache \
    go mod download && \
    go mod verify

COPY . ./

ARG TARGETARCH
RUN --mount=type=cache,target=/go/.cache \
    GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o health-proxy \
    ./main.go

FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.source=https://github.com/JSchlarb/health-proxy

WORKDIR /app

COPY --from=builder /app/health-proxy ./health-proxy

USER 65532:65532

ENTRYPOINT ["/app/health-proxy"]
