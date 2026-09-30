PREFIX ?= /usr/local
BINARY := vscan
PKG := ./cmd/vscan

.PHONY: build test install cross clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY) $(PKG)

test:
	CGO_ENABLED=0 go test ./...

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

cross:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/vscan-darwin-amd64 $(PKG)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/vscan-darwin-arm64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/vscan-linux-amd64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/vscan-linux-arm64 $(PKG)

clean:
	rm -rf bin dist
