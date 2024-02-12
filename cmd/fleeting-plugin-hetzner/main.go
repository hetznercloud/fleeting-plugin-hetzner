package main

import (
	"flag"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
	hetzner "gitlab.com/hiboxsystems/fleeting-plugin-hetzner"
)

var (
	showVersion = flag.Bool("version", false, "Show version information and exit")
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println(hetzner.Version.Full())
		os.Exit(0)
	}

	plugin.Serve(&hetzner.InstanceGroup{})
}
