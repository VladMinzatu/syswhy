# 0001: Embed Inspektor Gadget and its gadgets in the syswhy binary

Status: proposed (2026-10-23)

## Context

syswhy is a standalone TUI that runs eBPF diagnostics via Inspektor Gadget
(IG), and must also run gadgets of our own. IG gadgets are OCI images (eBPF
programs + metadata, optionally Wasm); IG's ~45 built-in gadgets (e.g.
`trace_exec`, `profile_cpu`, `top_*`) are images on ghcr.io. Beyond running
gadgets, IG provides user/kernel stack symbolization and container discovery
and enrichment (Docker, containerd, CRI-O, Podman), which we want to reuse.
IG is v0.x (v0.57.0, 2026-10-05) with no API stability promise.

Spikes on the dev setup (OrbStack, arm64, kernel 7.0) ran `trace_exec` via
the Go library and via the `ig` CLI, built and ran a custom hello-world
gadget through both, and ran gadgets embedded in the binary.

## Findings

Library vs CLI:

|                        | (a) Go library (`runtime/local`)            | (b) `ig run -o json` subprocess                    |
|------------------------|---------------------------------------------|----------------------------------------------------|
| Build                  | `CGO_ENABLED=0` works; 512 modules; must copy IG's 2 `replace` directives into our `go.mod` | Download release binary |
| Size                   | ~90 MB binary                               | 115 MB `ig` + our binary                           |
| Output                 | Datasources with typed field accessors and schema | JSON lines on stdout; no schema              |
| Enrichment             | Opt-in: we compose operators                | Built in                                           |
| Run requirements       | `make run` flags only                       | Also `-v /:/host`, or it fails                     |
| API churn              | Minimal path unchanged v0.45 -> v0.57       | CLI flags                                          |

Embedding (library, OCI layout via `//go:embed` + `orasoci.NewFromFS` +
`gadgetcontext.WithOrasReadonlyTarget`):

- Works fully offline (`--network none`): `trace_exec`, `trace_open`,
  `top_process`, and our custom gadget.
- Exported archives contain amd64 and arm64; IG picks the host's.
- Binary grows 1:1 with archive size. Measured: `trace_exec` 5.9 MB (2.6 MB
  Wasm + 1.6 MB eBPF per arch), `trace_open` 0.9 MB, `profile_cpu` 0.2 MB,
  `top_process` 0.02 MB, hello 0.1 MB. Four IG gadgets: 91.6 -> 98.7 MB.
- IG skips signature verification for explicitly provided targets: only
  images resolved through its local store are verified.
- Some gadgets are metadata driving Go operators compiled into the host
  binary (`top_process` uses only the `process` operator). They produce
  nothing unless that operator is compiled in, initialized with its global
  params, and passed to the gadget context.
- The library initializes no operators by default (hence the oci-handler's
  "verification disabled" warning in the first spike).

Custom gadgets: build Docker-only in a stage `FROM gadget-builder:<IG
version>` running `ig image build --local` (~30 s, no Docker-in-Docker), and
run exactly like built-ins; output is schema-driven, needing no
gadget-specific consumer code.

No OrbStack limitations found.

## Decision

Embed IG as a Go library, and embed all gadgets, IG's and ours, in the
binary as one OCI layout. All IG imports stay in `internal/adapters/ig`.

- One self-contained binary: no registry access, image store, or volume at
  runtime.
- Typed datasource access instead of parsing CLI output.
- Built-in and custom gadgets share one code path.

Build pipeline (Dockerfile stages):

1. Pull a pinned list of IG gadgets with the `ig` CLI, verifying signatures.
2. Build our gadgets from `gadgets/` with `gadget-builder`.
3. Export all into one OCI archive and unpack it as a layout.
4. Embed the layout via `//go:embed` and build the static binary.

## Consequences

- One `IG_VERSION` pins the Go module, `ig` CLI, gadget tags, and
  `gadget-builder`; upgrades bump all and re-copy IG's `replace` directives.
- Trust is established at build time (verified pull, pinned digests); the
  embedded gadgets are as trustworthy as our binary. Our own gadgets need no
  signing.
- The compiled-in operator set must match the shipped gadgets. An
  integration test runs every embedded gadget and expects output.
- We initialize operators with their global params and pass data operators
  explicitly.
- Building needs network access to ghcr.io; BuildKit caching keeps rebuilds
  fast. The embedded layout is generated and gitignored; unit tests use a
  stub.
- Binary size: ~90 MB base plus gadgets 1:1. Options if it matters later:
  strip the foreign architecture, or compress and unpack at startup.
- Large dependency tree: ~45 s cold builds, mitigated by BuildKit caches.

## Verified while building the pipeline

- `ig image pull` does not verify signatures; `ig image verify` does (it
  rejects unsigned images), so the build runs it after each pull. Pulling by
  digest stores the image untagged, so the build pulls by tag and compares
  the digest with the pin.
- IG's `host.Init` (called by the local runtime) calls `os.Exit` when debugfs
  or tracefs is missing. The adapter checks these first and returns an
  error; the readiness checks include debugfs.

Still open: which architectures `ig image build` produces for our gadgets,
and startup time with embedded gadgets.
