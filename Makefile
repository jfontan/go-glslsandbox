
ASSETS = $(shell find assets/ -type f)

.PHONY: assets
assets: assets.go

assets.go: $(ASSETS)
	esc -pkg="glsl" -o="assets.go" assets
