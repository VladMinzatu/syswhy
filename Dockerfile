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
    go vet ./... && go test -timeout 60s ./...

FROM base AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -trimpath -o /out/syswhy ./cmd/syswhy

# Integration test binaries, compiled here so the privileged container only
# runs them. The debug variant provides a shell to loop over them.
FROM base AS integration-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go test -c -tags integration -o /out/ ./...

FROM gcr.io/distroless/static-debian13:debug AS integration
COPY --from=integration-build /out/ /tests/
ENTRYPOINT ["/busybox/sh", "-c", "for t in /tests/*.test; do $t -test.run Integration -test.v || exit 1; done"]

# Runs as root: eBPF diagnostics need privileges on the host.
FROM gcr.io/distroless/static-debian13 AS runtime
COPY --from=build /out/syswhy /usr/local/bin/syswhy
ENTRYPOINT ["/usr/local/bin/syswhy"]
