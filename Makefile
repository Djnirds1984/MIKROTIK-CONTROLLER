APP_NAME = mikrotik-controller
BUILD_DIR = build

.PHONY: all build clean linux-arm linux-arm64 linux-amd64 run

all: build

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) .

run:
	go run .

clean:
	rm -rf $(BUILD_DIR)

# Cross-compilation targets for SBC boards
linux-arm:
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-arm .

linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

# Build for all platforms
release: clean linux-arm linux-arm64 linux-amd64
	@echo "Build complete. Binaries in $(BUILD_DIR)/"
