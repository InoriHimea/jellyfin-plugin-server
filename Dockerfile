# ── Stage 1: Build Web UI ─────────────────────────────────────────────────────
FROM node:20-alpine AS ui-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --prefer-offline
COPY web/ ./
RUN npm run build

# ── Stage 2: Build Go binary ──────────────────────────────────────────────────
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/web/dist ./internal/handler/dist

ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
      -X github.com/inorihimea/jellyfin-plugin-server/internal/handler.Version=${VERSION} \
      -X github.com/inorihimea/jellyfin-plugin-server/internal/handler.GitCommit=${GIT_COMMIT}" \
    -o /bin/jellyfin-plugin-server ./cmd/server

# ── Stage 3: Minimal runtime image ────────────────────────────────────────────
FROM scratch
# CA certs for TLS upstream connections
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /bin/jellyfin-plugin-server /jellyfin-plugin-server

VOLUME ["/data"]
ENV JPSERVER_DATA_DIR=/data \
    JPSERVER_LOG_JSON=true
EXPOSE 8080
ENTRYPOINT ["/jellyfin-plugin-server"]
