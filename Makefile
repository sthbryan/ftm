.PHONY: build run clean test install release build-web tauri tauri-dev

VERSION := 0.4.1
BINARY := ftm
CMD := ./cmd/ftm
TAURI_DIR := desktop/src-tauri

build-web:
	cd web-svelte && bun install && bun run build
	rm -rf internal/web/static/*
	cp -r web-svelte/dist/* internal/web/static/
	touch internal/web/static/.gitkeep

build: build-web
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) $(CMD)

tauri: build-web
	cd $(TAURI_DIR) && cargo build --release

tauri-dev: build-web
	cd $(TAURI_DIR) && cargo run

run:
	go run $(CMD)

clean:
	rm -f $(BINARY) ftm-*
	cd $(TAURI_DIR) && cargo clean

test:
	go test ./...

install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY)

release:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-darwin-arm64 $(CMD)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-linux-amd64 $(CMD)
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)-windows.exe $(CMD)

dev:
	go run $(CMD)

fmt:
	go fmt ./...

vet:
	go vet ./...
