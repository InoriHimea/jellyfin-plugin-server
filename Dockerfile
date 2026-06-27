# ── Stage 1: Build Web UI ─────────────────────────────────────────────────────
# Always run on the build host (amd64) — no emulation needed for JS.
FROM --platform=$BUILDPLATFORM node:20-alpine AS ui-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --prefer-offline
COPY web/ ./
RUN npm run build

# ── Stage 2: Build Go binary ──────────────────────────────────────────────────
# Run on the build host (amd64) and cross-compile to TARGETARCH.
# This avoids QEMU emulation which makes arm64 builds 20× slower.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/web/dist ./internal/handler/dist

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w \
      -X github.com/inorihimea/jellyfin-plugin-server/internal/handler.Version=${VERSION} \
      -X github.com/inorihimea/jellyfin-plugin-server/internal/handler.GitCommit=${GIT_COMMIT}" \
    -o /bin/jellyfin-plugin-server ./cmd/server

# ── Stage 3: Minimal runtime image ────────────────────────────────────────────
FROM scratch
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /bin/jellyfin-plugin-server /jellyfin-plugin-server

VOLUME ["/data"]
ENV JPSERVER_DATA_DIR=/data \
    JPSERVER_LOG_JSON=true
EXPOSE 8080
ENTRYPOINT ["/jellyfin-plugin-server"]
