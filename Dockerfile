# syntax=docker/dockerfile:1.7

ARG GO_IMAGE=golang:1.26.5-bookworm@sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599
ARG RUNTIME_IMAGE=debian:bookworm-slim@sha256:abd67ffcfa541b485a3dff59865ab629aa048a6c613e639d36e7456b0b229241

FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS build
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    test -n "$TARGETOS" && test -n "$TARGETARCH" && \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /out/jetkvm-mcp ./cmd/jetkvm-mcp

FROM scratch AS binary
COPY --from=build /out/jetkvm-mcp /jetkvm-mcp

FROM ${RUNTIME_IMAGE}
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 jetkvm \
    && useradd --system --uid 10001 --gid jetkvm --home-dir /nonexistent --shell /usr/sbin/nologin jetkvm
COPY --from=build /out/jetkvm-mcp /usr/local/bin/jetkvm-mcp
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/jetkvm-mcp"]
