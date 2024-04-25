package main

import (
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
	hetzner "gitlab.com/hiboxsystems/fleeting-plugin-hetzner"
)

func main() {
	plugin.Main(&hetzner.InstanceGroup{}, hetzner.Version)
}
