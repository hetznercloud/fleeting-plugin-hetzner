package main

import (
	"flag"
	"fmt"
	"os"

	aws "gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws"
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
)

var (
	showVersion = flag.Bool("version", false, "Show version information and exit")
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println(aws.Version.Full())
		os.Exit(0)
	}

	plugin.Serve(&aws.InstanceGroup{})
}
