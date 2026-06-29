# Build/run Tsugi with version metadata stamped from git.
BINARY := tsugi
PKG    := github.com/latoulicious/Tsugi
CMD    := ./cmd/tsugi
IMAGE  := tsugi
PREFIX ?= /usr/local/bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

.PHONY: build run test vet tidy proto image install

# Regenerate the gRPC stubs from the vendored contract (source of truth: Yagura).
# Needs protoc + protoc-gen-go + protoc-gen-go-grpc on PATH. Generated *.pb.go are
# committed so the Docker build stays protoc-free.
proto:
	protoc --go_out=. --go_opt=module=$(PKG) \
		--go-grpc_out=. --go-grpc_opt=module=$(PKG) \
		proto/tsugi_agent.proto

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) $(CMD)

run: build
	./bin/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

image:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t $(IMAGE):$(VERSION) .

# Extract the static binary from the image (the box has no Go toolchain).
install: image
	cid=$$(docker create $(IMAGE):$(VERSION)) && \
	docker cp $$cid:/usr/local/bin/$(BINARY) $(PREFIX)/$(BINARY) && \
	docker rm $$cid
