# Variables
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin
BINARY = brown_noise

.PHONY: all build install uninstall clean

all: build

build:
	@echo "Building Go binary..."
	go build -o $(BINARY) .

install: build
	@echo "Installing $(BINARY) to $(BINDIR)..."
	install -Dm755 $(BINARY) $(BINDIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(BINDIR)/$(BINARY)"

uninstall:
	@echo "Uninstalling $(BINARY) from $(BINDIR)..."
	-rm -f $(BINDIR)/$(BINARY)
	@echo "Uninstalled."

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY)