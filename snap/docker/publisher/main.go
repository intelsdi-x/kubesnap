package main

import (
	"os"

	"github.com/intelsdi-x/snap/control/plugin"
	//"marcintao/goworks/api/publisher"
	//"marcintao/goworks/api/server"
	"github.com/intelsdi-x/kubesnap/snap/docker/publisher/publisher"
)

func main() {
	meta := publisher.Meta()
	if publisherCore, err := publisher.NewCore(); err != nil {
		panic(err)
	} else {
		plugin.Start(meta, publisherCore, os.Args[1])
	}
}
