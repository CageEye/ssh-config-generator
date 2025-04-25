all: install

build: check_go
	go build -o cageeyessh

deploy: build
	cp cageeyessh ~/.local/bin/cageeyessh

include:
	@if ! grep -q "Include prod.config" ~/.ssh/config; then \
		echo "Include prod.config" >> ~/.ssh/config; \
	fi

install: include deploy
	@cp prod.config ~/.ssh/prod.config
	@cp staging.config ~/.ssh/staging.config

check_go:
	@if [ -z "$$(command -v go)" ]; then \
		echo "Error: Go is not installed or not in PATH"; \
		exit 1;\
	fi

