.PHONY: all build build-all macos-arm64 darwin-arm64 debian-arm64 linux-arm64 ubuntu-amd64 linux-amd64 package clean help

# Default target
all: build-all

help:
	@echo "Keet ADK Gateway Build Commands:"
	@echo "  make build          - Build for current host platform"
	@echo "  make build-all      - Build for macOS (Apple Silicon), Debian (ARM64), and Ubuntu (AMD64)"
	@echo "  make macos-arm64    - Build for macOS Apple Silicon (darwin/arm64)"
	@echo "  make debian-arm64   - Build for Debian Linux ARM64 (linux/arm64)"
	@echo "  make ubuntu-amd64   - Build for Ubuntu AMD64 (linux/amd64)"
	@echo "  make package        - Build and create release tarballs with SHA256 checksums"
	@echo "  make clean          - Remove built binaries and distribution packages"

build:
	@./build.sh host

build-all:
	@./build.sh all

macos-arm64 darwin-arm64:
	@./build.sh macos-arm64

debian-arm64 linux-arm64:
	@./build.sh debian-arm64

ubuntu-amd64 linux-amd64:
	@./build.sh ubuntu-amd64

package:
	@./build.sh package

clean:
	@./build.sh clean
