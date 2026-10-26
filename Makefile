IMAGE ?= syswhy:dev

# What eBPF diagnostics need from the host.
RUN_FLAGS = --privileged --pid=host \
	-v /sys/fs/bpf:/sys/fs/bpf \
	-v /sys/kernel/debug:/sys/kernel/debug \
	-v /sys/kernel/tracing:/sys/kernel/tracing

.PHONY: test build run

test:
	docker build --target test --output type=cacheonly .

build:
	docker build --target runtime -t $(IMAGE) .

run: build
	docker run --rm -it $(RUN_FLAGS) $(IMAGE)
