# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27.3
ARG ALPINE_VERSION=3.24.2

# Toolchain + downloaded modules. The Inspektor Gadget (IG) version is taken
# from go.mod, so the Go module, ig CLI and gadget images stay in lockstep.
FROM golang:${GO_VERSION} AS base
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download \
 && go list -m -f '{{.Version}}' github.com/inspektor-gadget/inspektor-gadget > /ig-version

# Gadgets embedded in the binary, as an OCI layout at /oci.
FROM alpine:${ALPINE_VERSION} AS gadgets
ARG BUILDARCH
COPY --from=base /ig-version /ig-version
RUN IG_VERSION=$(cat /ig-version) \
 && wget -qO- https://github.com/inspektor-gadget/inspektor-gadget/releases/download/${IG_VERSION}/ig-linux-${BUILDARCH}-${IG_VERSION}.tar.gz \
    | tar xz -C /usr/local/bin ig
COPY gadgets/ /gadgets/
RUN IG_VERSION=$(cat /ig-version) /gadgets/build.sh /oci

# Unit tests; cached by BuildKit while sources are unchanged. They don't need
# the embedded gadgets.
FROM base AS test
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    go vet ./... && go test -timeout 60s ./...

FROM base AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    --mount=type=bind,from=gadgets,source=/oci,target=internal/adapters/ig/oci \
    CGO_ENABLED=0 go build -trimpath -o /out/syswhy ./cmd/syswhy

# Integration test binaries, compiled here so the privileged container only
# runs them. The debug variant provides a shell to loop over them.
FROM base AS integration-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    --mount=type=bind,from=gadgets,source=/oci,target=internal/adapters/ig/oci \
    CGO_ENABLED=0 go test -c -tags integration -o /out/ ./...

FROM gcr.io/distroless/static-debian13:debug AS integration
COPY --from=integration-build /out/ /tests/
ENTRYPOINT ["/busybox/sh", "-c", "for t in /tests/*.test; do $t -test.run Integration -test.v || exit 1; done"]

# Runs as root: eBPF diagnostics need privileges on the host.
FROM gcr.io/distroless/static-debian13 AS runtime
COPY --from=build /out/syswhy /usr/local/bin/syswhy
ENTRYPOINT ["/usr/local/bin/syswhy"]
