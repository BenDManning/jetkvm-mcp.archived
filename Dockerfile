# syntax=docker/dockerfile:1.7.1@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

FROM debian:trixie-slim@sha256:3a39a0592364683e6bab97937b72cad5a8fa6dcbbee90edb3bb48c7f8e94f258 AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 jetkvm \
    && useradd --system --uid 10001 --gid jetkvm --home-dir /nonexistent --shell /usr/sbin/nologin jetkvm \
    && install -d -m 0755 /usr/share/jetkvm-mcp \
    && dpkg-query --show --showformat='${binary:Package}\t${Version}\n' ca-certificates ffmpeg \
        > /usr/share/jetkvm-mcp/container-packages.txt \
    && ffmpeg -version > /usr/share/jetkvm-mcp/ffmpeg-version.txt

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
ARG SOURCE=https://github.com/BenDManning/jetkvm-mcp
ARG REVISION=unknown
ARG VERSION=dev
ARG CREATED=1970-01-01T00:00:00Z
LABEL org.opencontainers.image.source=$SOURCE \
      org.opencontainers.image.revision=$REVISION \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.licenses=MIT \
      org.opencontainers.image.created=$CREATED
COPY --from=build /out/jetkvm-mcp /usr/local/bin/jetkvm-mcp
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/jetkvm-mcp"]
