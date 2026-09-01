# syntax=docker/dockerfile:1.7.1@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

FROM debian:trixie-slim@sha256:d7e12182ce18b85b93007c1dedf31f2d29e01ccf3182cc4017c709b6259bc132 AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 jetkvm \
    && useradd --system --uid 10001 --gid jetkvm --home-dir /nonexistent --shell /usr/sbin/nologin jetkvm \
    && install -d -m 0755 /usr/share/jetkvm-mcp \
    && dpkg-query --show --showformat='${binary:Package}\t${Version}\n' ca-certificates ffmpeg \
        > /usr/share/jetkvm-mcp/container-packages.txt \
    && ffmpeg -version > /usr/share/jetkvm-mcp/ffmpeg-version.txt

FROM --platform=$BUILDPLATFORM golang:1.27.0-bookworm@sha256:ded31c68586d2e49e760acc2e65a884b23d032e9bbbed0ae0c55abd3fcaf4452 AS build
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
