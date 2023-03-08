package main

import (
	"fmt"

	aws "gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws"
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
)

func main() {
	fmt.Println(aws.Version.String())
	plugin.Serve(&aws.InstanceGroup{})
}
