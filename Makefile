.PHONY: build prepare-web dev-api dev-web test clean

CONFIG ?= ../configs/mdfs.dev.toml
GOCACHE ?= $(CURDIR)/.cache/go-build
export GOCACHE

build:
	$(MAKE) prepare-web test
	goreleaser release --snapshot --clean

prepare-web:
	./scripts/prepare-web.sh

dev-api:
	cd backend && go run ./cmd/mdfs --config $(CONFIG)

dev-web:
	cd frontend && npm run dev

test:
	cd backend && go test ./...
	cd frontend && npm test
	cd frontend && npm run typecheck
	cd frontend && npm run build

clean:
	rm -rf dist frontend/dist backend/internal/webui/dist
	mkdir -p backend/internal/webui/dist
	printf '%s\n' 'The frontend production build is copied into this directory by scripts/prepare-web.sh.' > backend/internal/webui/dist/README.txt
