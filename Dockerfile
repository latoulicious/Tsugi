FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

# Build identity injected at link time; defaults match the version package.
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w \
    -X github.com/latoulicious/Tsugi/internal/version.Version=${VERSION} \
    -X github.com/latoulicious/Tsugi/internal/version.Commit=${COMMIT} \
    -X github.com/latoulicious/Tsugi/internal/version.Date=${DATE}" \
  -o /out/tsugi ./cmd/tsugi

FROM alpine:3.23

RUN adduser -D -H tsugi

COPY --from=build /out/tsugi /usr/local/bin/tsugi

USER tsugi
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

CMD ["tsugi"]
