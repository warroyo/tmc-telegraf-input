GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOARM ?= $(shell go env GOARM)

.PHONY: all
all: tmcevents

.PHONY: tmcevents
tmcevents:
	go build ./cmd/main.go

.PHONY: dist
dist: tmcevents-$(GOOS)-$(GOARCH).tar.gz

.PHONY: dist-all
dist-all: tmcevents-linux-amd64.tar.gz
dist-all: tmcevents-windows-amd64.zip
dist-all: tmcevents-darwin-amd64.tar.gz

tmcevents-linux-amd64.tar.gz: GOOS := linux
tmcevents-linux-amd64.tar.gz: GOARCH := amd64
tmcevents-windows-amd64.zip: GOOS := windows
tmcevents-windows-amd64.zip: GOARCH := amd64
tmcevents-windows-amd64.zip: EXT := .exe
tmcevents-darwin-amd64.zip: GOOS := darwin
tmcevents-darwin-amd64.zip: GOARCH := amd64
tmcevents-%.tar.gz:
	mkdir -p "dist/tmcevents-$*"
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o "dist/tmcevents-$*/tmcevents$(EXT)" -ldflags "-w -s" ./cmd/main.go
	cp tmcevents.conf "dist/tmcevents-$*"
	cd dist && tar czf "$@" "tmcevents-$*"

tmcevents-%.zip:
	mkdir -p "dist/tmcevents-$*"
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o "dist/tmcevents-$*/tmcevents$(EXT)" -ldflags "-w -s" ./cmd/main.go
	cp tmcevents.conf "dist/tmcevents-$*"
	cd dist && zip -r "$@" "tmcevents-$*"

.PHONY: clean
clean:
	rm -f main{,.exe}
	rm -rf dist