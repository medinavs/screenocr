APP_NAME := screenocr
VERSION  := 1.0.0
BUILD_DIR := build

WINDOWS_FLAGS := -trimpath -ldflags="-s -w -H windowsgui -X main.Version=$(VERSION)"
GO_FLAGS      := -trimpath -ldflags="-s -w -X main.Version=$(VERSION)"

.PHONY: all clean build run test lint winres icon \
        build-windows build-linux build-darwin build-darwin-arm \
        install-deps-linux install-deps-macos install-deps-windows

all: build-windows

deps:
	go mod tidy
	go mod download

# windows resource file (.syso) — embeds icon + manifest
winres:
	go run github.com/tc-hib/go-winres@latest make --in winres/winres.json --out cmd/screenocr/rsrc

icon:
	go run winres/gen_icon.go
	go run winres/gen_tray_icon.go

build-windows: deps winres
	@mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
		go build $(WINDOWS_FLAGS) -o $(BUILD_DIR)/windows/$(APP_NAME).exe ./cmd/screenocr

# linux (systray + hotkey require CGO)
build-linux: deps
	@mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
		go build $(GO_FLAGS) -o $(BUILD_DIR)/linux/$(APP_NAME) ./cmd/screenocr

# macOS amd64 (systray + hotkey require CGO)
build-darwin: deps
	@mkdir -p $(BUILD_DIR)/darwin
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		go build $(GO_FLAGS) -o $(BUILD_DIR)/darwin/$(APP_NAME) ./cmd/screenocr

# macOS arm64 (Apple Silicon)
build-darwin-arm: deps
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		go build $(GO_FLAGS) -o $(BUILD_DIR)/darwin-arm64/$(APP_NAME) ./cmd/screenocr


run: deps
	go run ./cmd/screenocr -verbose

run-notray: deps
	go run ./cmd/screenocr -verbose -no-tray

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)


install-deps-windows:
	@echo "1. Install Tesseract: https://github.com/UB-Mannheim/tesseract/wiki"
	@echo "2. Add to PATH: C:\\Program Files\\Tesseract-OCR"
	@echo "3. Open a new terminal and verify: tesseract --version"
	@echo "4. Build: make build-windows  (CGO not required)"

install-deps-linux:
	sudo apt-get update
	sudo apt-get install -y \
		tesseract-ocr tesseract-ocr-eng \
		libx11-dev libxrandr-dev \
		libnotify-dev slop \
		gcc

install-deps-macos:
	brew install tesseract pkg-config
