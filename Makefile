BINARY := uauth
PREFIX := $(HOME)/.local/bin

.PHONY: build test install clean

build:
	go build -o $(BINARY) .

test:
	go test -v ./...

install: build
	mkdir -p $(PREFIX)
	cp $(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed to $(PREFIX)/$(BINARY)"

clean:
	rm -f $(BINARY)
