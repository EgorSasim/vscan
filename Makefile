PREFIX ?= /usr/local
BINARY := vscan
PKG := ./cmd/vscan

.PHONY: build build-response test install cross clean

# -tags noresponse keeps internal/response out of the binary (VPS build).
build:
	CGO_ENABLED=0 go build -tags noresponse -trimpath -ldflags="-s -w" -o bin/$(BINARY) $(PKG)

build-response:
	CGO_ENABLED=0 go build -tags response -trimpath -ldflags="-s -w" -o bin/vscan-response $(PKG)

test:
	CGO_ENABLED=0 go test -tags noresponse ./...
	CGO_ENABLED=0 go test -tags response ./...

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 bin/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

cross:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags noresponse -trimpath -ldflags="-s -w" -o dist/vscan-darwin-amd64 $(PKG)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags noresponse -trimpath -ldflags="-s -w" -o dist/vscan-darwin-arm64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags noresponse -trimpath -ldflags="-s -w" -o dist/vscan-linux-amd64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags noresponse -trimpath -ldflags="-s -w" -o dist/vscan-linux-arm64 $(PKG)

clean:
	rm -rf bin dist
