package main

import (
	"github.com/src-d/go-cli"
)

var (
	version string
	build   string
)

var app = cli.New("glsl", version, build, "glslsandbox is a fragment shader editor and gallery for the web")

func main() {
	app.RunMain()
}
