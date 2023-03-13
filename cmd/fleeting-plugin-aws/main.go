package main

import (
	aws "gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws"
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
)

func main() {
	plugin.Serve(&aws.InstanceGroup{})
}
