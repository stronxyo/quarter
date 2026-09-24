package main

import (
	"flag"
	"fmt"

	"github.com/stronxyo/quarter/internal/config"
)

// version is overridden at release time with -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s %s\n", config.AppName, version)
		return
	}

	fmt.Printf("%s: nothing to show yet. Try --version.\n", config.AppName)
}
