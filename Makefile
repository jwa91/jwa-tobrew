.PHONY: build install dev check release clean

BIN_DIR ?= $(HOME)/.local/bin

build:
	go build -o bin/jwa-tobrew ./cmd/jwa-tobrew

install: build
	install -d $(BIN_DIR)
	install -m 0755 bin/jwa-tobrew $(BIN_DIR)/jwa-tobrew
	@echo "installed jwa-tobrew to $(BIN_DIR)/jwa-tobrew"
	@echo "ensure $(BIN_DIR) is on your PATH"

dev: install

check:
	go vet ./...
	go test ./...

# Local release. CI handles tag pushes automatically; this target is
# only for ad-hoc local cuts. Requires 1Password signed in and the
# v$(VERSION) tag already pointing at HEAD.
release:
	@test -n "$(VERSION)" || (echo "usage: make release VERSION=X.Y.Z" && exit 2)
	@op whoami >/dev/null || (echo "1Password not signed in: eval \$$(op signin)" && exit 1)
	@existing=$$(git rev-parse -q --verify "v$(VERSION)^{commit}" 2>/dev/null); \
	head=$$(git rev-parse HEAD); \
	test -n "$$existing" && test "$$existing" = "$$head" || \
	  (echo "v$(VERSION) must exist and point at HEAD before release"; exit 3)
	jwa-harden run -- goreleaser release --clean

clean:
	rm -rf bin dist
