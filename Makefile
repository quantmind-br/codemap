.PHONY: all build build-mcp run deps grammars clean install install-mcp uninstall

all: build

build:
	go build -o codemap .

build-mcp:
	go build -o codemap-mcp ./mcp/

DIR ?= .
ABS_DIR := $(shell cd "$(DIR)" && pwd)
SKYLINE_FLAG := $(if $(SKYLINE),--skyline,)
ANIMATE_FLAG := $(if $(ANIMATE),--animate,)
DEPS_FLAG := $(if $(DEPS),--deps,)

run: build
	./codemap $(SKYLINE_FLAG) $(ANIMATE_FLAG) $(DEPS_FLAG) "$(ABS_DIR)"

# Build tree-sitter grammar libraries (one-time setup for deps mode)
grammars:
	cd scanner && ./build-grammars.sh

# Dependency graph mode - shows functions and imports per file
deps: build grammars
	./codemap --deps "$(ABS_DIR)"

clean:
	rm -f codemap codemap-mcp
	rm -rf scanner/.grammar-build
	rm -rf scanner/grammars

# Installation paths
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
GRAMMAR_DIR ?= $(PREFIX)/lib/codemap/grammars

install: build
	@echo "Installing codemap to $(BINDIR)..."
	install -d $(BINDIR)
	install -m 755 codemap $(BINDIR)/codemap
	@if [ -d scanner/grammars ] && [ "$$(ls -A scanner/grammars 2>/dev/null)" ]; then \
		echo "Installing grammars to $(GRAMMAR_DIR)..."; \
		install -d $(GRAMMAR_DIR); \
		cp -r scanner/grammars/* $(GRAMMAR_DIR)/; \
	fi
	@echo "Done! Run 'codemap --help' to get started."

install-mcp: build-mcp
	@echo "Installing codemap-mcp to $(BINDIR)..."
	install -d $(BINDIR)
	install -m 755 codemap-mcp $(BINDIR)/codemap-mcp
	@echo "Done!"

uninstall:
	@echo "Removing codemap from $(BINDIR)..."
	rm -f $(BINDIR)/codemap
	rm -f $(BINDIR)/codemap-mcp
	rm -rf $(PREFIX)/lib/codemap
	@echo "Done!"
