.PHONY: build run dev install uninstall test fmt vet clean release snapshot

VERSION ?= dev
LDFLAGS := -s -w -X github.com/dockttier/dockttier/cmd.version=$(VERSION)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/dockttier .

dev:
	go run . $(ARGS)

install: build
	sudo install -m 0755 bin/dockttier /usr/local/bin/dockttier
	sudo ln -sf /usr/local/bin/dockttier /usr/local/bin/docker
	@echo "Installed. Run 'hash -r' (bash) / 'rehash' (zsh) or open a new shell."

uninstall:
	sudo update-alternatives --remove docker /usr/local/bin/dockttier 2>/dev/null || true
	@if [ -L /usr/local/bin/docker ]; then sudo rm -f /usr/local/bin/docker; fi
	sudo rm -f /usr/local/bin/dockttier
	@echo "Uninstalled. Run 'hash -r' / 'rehash' or open a new shell."

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin dist

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean
