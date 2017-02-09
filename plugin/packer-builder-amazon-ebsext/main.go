package main

import (
	"github.com/dnabic/packer-ebsext"
	"github.com/mitchellh/packer/packer/plugin"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(ebsext.Builder))
	server.Serve()
}
