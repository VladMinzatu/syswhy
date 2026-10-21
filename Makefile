IMAGE ?= syswhy:dev

.PHONY: test build run

test:
	docker build --target test --output type=cacheonly .

build:
	docker build --target runtime -t $(IMAGE) .

run: build
	docker run --rm -it $(IMAGE)
