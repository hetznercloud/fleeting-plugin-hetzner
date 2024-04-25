package hetzner

import (
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
)

var (
	NAME      = "fleeting-plugin-hetzner"
	VERSION   = "dev"
	REVISION  = "HEAD"
	REFERENCE = "HEAD"
	BUILT     = "now"

	Version plugin.VersionInfo
)

func init() {
	Version = plugin.VersionInfo{
		Name:      NAME,
		Version:   VERSION,
		Revision:  REVISION,
		Reference: REFERENCE,
		BuiltAt:   BUILT,
	}
}
