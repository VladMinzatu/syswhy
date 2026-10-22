IMAGE ?= syswhy:dev
IT_IMAGE ?= syswhy:integration

# What eBPF diagnostics need from the host.
RUN_FLAGS = --privileged --pid=host \
	-v /sys/fs/bpf:/sys/fs/bpf \
	-v /sys/kernel/debug:/sys/kernel/debug \
	-v /sys/kernel/tracing:/sys/kernel/tracing

.PHONY: test test-integration build run

test:
	docker build --target test --output type=cacheonly .

# Runs against the real kernel, both as `make run` would and unprivileged.
test-integration:
	docker build --target integration -t $(IT_IMAGE) .
	docker run --rm $(RUN_FLAGS) -e SYSWHY_IT_ENV=privileged $(IT_IMAGE)
	docker run --rm -e SYSWHY_IT_ENV=unprivileged $(IT_IMAGE)

build:
	docker build --target runtime -t $(IMAGE) .

run: build
	docker run --rm -it $(RUN_FLAGS) $(IMAGE)
