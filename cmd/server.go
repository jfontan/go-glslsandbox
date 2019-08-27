package main

import (
	glsl "github.com/jfontan/go-glslsandbox"
	"github.com/src-d/go-cli"
)

func init() {
	app.AddCommand(&serverCommand{})
}

type serverCommand struct {
	cli.Command `name:"server" short-description:"start web service"`
}

func (i *serverCommand) Execute(args []string) error {
	db, err := prepareDB()
	if err != nil {
		return err
	}
	defer db.Close()

	server := glsl.NewServer(db, true)
	server.Start()
	return nil
}
