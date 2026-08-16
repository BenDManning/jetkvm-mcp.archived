# syntax=docker/dockerfile:1.7.1@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

FROM debian:bookworm-slim@sha256:abd67ffcfa541b485a3dff59865ab629aa048a6c613e639d36e7456b0b229241 AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 jetkvm \
    && useradd --system --uid 10001 --gid jetkvm --home-dir /nonexistent --shell /usr/sbin/nologin jetkvm

FROM --platform=$BUILDPLATFORM golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36 AS build
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

FROM runtime
COPY --from=build /out/jetkvm-mcp /usr/local/bin/jetkvm-mcp
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/jetkvm-mcp"]
