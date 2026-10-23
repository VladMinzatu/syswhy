# syswhy

Interactive TUI for performance diagnostics on a single Linux node, built on
Inspektor Gadget (IG). Go + Bubble Tea.

## Working agreement

- **Never commit, push, or create branches.** Leave all changes uncommitted in
  the working tree; the user reviews, commits, and pushes.
- Work in **vertical slices**: one user-visible capability, end to end. A slice
  may span several **phases**; each phase is one small, reviewable change that
  builds and passes tests on its own. Stop after each phase for review.
- Plan before implementing a slice; keep the plan short.
- The process is deliberately minimal and evolves. Propose changes to this file
  when something causes friction, rather than adding process up front.

## Architecture (hexagonal)

```
cmd/syswhy/        wiring only: build adapters, inject into app, start TUI
internal/domain/   pure types and logic; no IG, TUI, or OS imports
internal/app/      use cases; defines ports (interfaces) it needs
internal/adapters/ implementations of ports (e.g. host/, ig/) and driving
                   adapters (tui/)
```

- Dependencies point inward: adapters -> app -> domain. Never the reverse.
- Ports are defined by the consumer (in `app`), kept small.
- Every port gets a fake for tests.

## Tests

- Everything testable gets tests, written alongside the code in the same phase.
- domain/app: plain unit tests using fakes.
- TUI: test the Bubble Tea model's `Update`/`View` directly.
- Adapters touching the kernel/host: integration tests run in the privileged
  container.

## Build & run (Docker only)

Dev target for now: arm64, Docker via OrbStack on macOS. The product targets
real modern Linux hosts: never design around OrbStack limitations; flag them
explicitly instead.

Known OrbStack limitations:

- `--pid=host` gives Docker's PID namespace, which is nested: eBPF reports
  PIDs from the kernel's initial namespace, so they don't match `/proc` in
  the container. Tests checking PIDs run only in the initial namespace.

No host Go toolchain is assumed. All builds and tests run in Docker; the
Makefile is a thin wrapper over `docker build`/`docker run`.

- `make test` - run tests (Dockerfile `test` stage)
- `make build` - build the runtime image
- `make run` - run the TUI on the current Docker host (privileged)
