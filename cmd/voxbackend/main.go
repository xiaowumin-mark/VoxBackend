package main

import (
	"log"
	"os"

	"github.com/xiaowumin-mark/VoxBackend/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdin); err != nil {
		log.Fatal(err)
	}
}
