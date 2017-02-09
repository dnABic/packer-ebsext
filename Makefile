lint:
	golint

build:
	@sh -c "$(CURDIR)/scripts/build.sh"

dev:
	@TF_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

style:
	gofmt -w .

test:
	go test

.PHONY: lint build dev style test 
