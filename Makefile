.PHONY: build run dev install uninstall test fmt vet clean release snapshot

VERSION ?= dev
LDFLAGS := -s -w -X github.com/onedord1/dockttier/cmd.version=$(VERSION)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/dockttier .

dev:
	go run . $(ARGS)

install: build
	sudo cp bin/dockttier /usr/local/bin/dockttier
	sudo update-alternatives --install /usr/bin/docker docker /usr/local/bin/dockttier 100

uninstall:
	sudo update-alternatives --remove docker /usr/local/bin/dockttier || true
	sudo rm -f /usr/local/bin/dockttier

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
