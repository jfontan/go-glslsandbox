
ASSETS = $(shell find assets/ -type f)

all:

.PHONY: assets
assets: assets.go

assets.go: $(ASSETS)
	esc -pkg="glsl" -o="assets.go" assets

devel: effects.db

effects.db:
	curl -O https://downloads.zooloo.org/glsl-devel.tar.gz
	tar xf glsl-devel.tar.gz

