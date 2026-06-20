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

.PHONY: build run test vet tidy image install

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
