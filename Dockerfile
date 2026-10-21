# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27.3

# Toolchain + downloaded modules.
FROM golang:${GO_VERSION} AS base
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

# Unit tests; cached by BuildKit while sources are unchanged.
FROM base AS test
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    go vet ./... && go test ./...

FROM base AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -trimpath -o /out/syswhy ./cmd/syswhy

# Runs as root: eBPF diagnostics need privileges on the host.
FROM gcr.io/distroless/static-debian13 AS runtime
COPY --from=build /out/syswhy /usr/local/bin/syswhy
ENTRYPOINT ["/usr/local/bin/syswhy"]
