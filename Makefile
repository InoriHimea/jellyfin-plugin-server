BINARY     := jellyfin-plugin-server
VERSION    := v1.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS    := -ldflags "-s -w \
	-X github.com/inorihimea/jellyfin-plugin-server/internal/handler.Version=$(VERSION) \
	-X github.com/inorihimea/jellyfin-plugin-server/internal/handler.GitCommit=$(GIT_COMMIT)"

.PHONY: build ui run test lint clean tidy

ui:
	cd web && npm run build
	rm -rf internal/handler/dist
	cp -r web/dist internal/handler/

build: ui
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

run: build
	./bin/$(BINARY)

dev-ui:
	cd web && npm run dev

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/ data/jellyfin.db internal/handler/dist
