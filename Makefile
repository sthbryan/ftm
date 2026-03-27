.PHONY: build run clean test install release build-web wails wails-dev desktop desktop-package desktop-all desktop-all-full desktop-darwin-universal desktop-darwin-arm64 desktop-darwin-amd64 desktop-linux-amd64 desktop-windows-amd64 dev fmt vet

VERSION := 0.7.0
BINARY := ftm
CMD := ./cmd/ftm
DESKTOP_DIR := ./desktop

build-web:
	./scripts/build-web-assets.sh
	mkdir -p $(DESKTOP_DIR)/build
	cp $(DESKTOP_DIR)/icon.png $(DESKTOP_DIR)/build/appicon.png

build: build-web
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) $(CMD)

wails: build-web
	cd $(DESKTOP_DIR) && wails build -s

wails-dev: build-web
	cd $(DESKTOP_DIR) && wails dev

desktop: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage

desktop-package: build-web
	cd $(DESKTOP_DIR) && wails build -s

desktop-darwin-universal: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform darwin/universal

desktop-darwin-arm64: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform darwin/arm64

desktop-darwin-amd64: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform darwin/amd64

desktop-linux-amd64: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform linux/amd64

desktop-windows-amd64: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform windows/amd64

desktop-all: build-web
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform darwin/universal
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform linux/amd64
	cd $(DESKTOP_DIR) && wails build -s -nopackage -platform windows/amd64

run:
	go run $(CMD)

clean:
	rm -f $(BINARY) ftm-*
	rm -rf $(DESKTOP_DIR)/build/bin/*
	rm -rf $(DESKTOP_DIR)/frontend/dist

test:
	go test ./...

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY)

release:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-darwin-arm64 $(CMD)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-linux-amd64 $(CMD)
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-linux-arm64 $(CMD)
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-windows.exe $(CMD)

dev:
	go run $(CMD)

fmt:
	go fmt ./...

vet:
	go vet ./...