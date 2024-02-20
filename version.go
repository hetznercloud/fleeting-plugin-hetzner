package hetzner

import (
	"fmt"
	"runtime"
)

var (
	NAME      = "fleeting-plugin-hetzner"
	VERSION   = "dev"
	REVISION  = "HEAD"
	REFERENCE = "HEAD"
	BUILT     = "now"

	Version VersionInfo
)

type VersionInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Revision     string `json:"revision"`
	Reference    string `json:"reference"`
	GOVersion    string `json:"go_version"`
	BuiltAt      string `json:"built_at"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

func (v *VersionInfo) String() string {
	return v.Version
}

func (v *VersionInfo) BuildInfo() string {
	return fmt.Sprintf(
		"sha=%s; ref=%s; go=%s; built_at=%s; os_arch=%s/%s",
		v.Revision,
		v.Reference,
		v.GOVersion,
		v.BuiltAt,
		v.OS,
		v.Architecture,
	)
}

func (v *VersionInfo) Full() string {
	version := fmt.Sprintf("Name:         %s\n", v.Name)
	version += fmt.Sprintf("Version:      %s\n", v.Version)
	version += fmt.Sprintf("Git revision: %s\n", v.Revision)
	version += fmt.Sprintf("Git ref:      %s\n", v.Reference)
	version += fmt.Sprintf("GO version:   %s\n", v.GOVersion)
	version += fmt.Sprintf("Built:        %s\n", v.BuiltAt)
	version += fmt.Sprintf("OS/Arch:      %s/%s\n", v.OS, v.Architecture)

	return version
}

func init() {
	Version = VersionInfo{
		Name:         NAME,
		Version:      VERSION,
		Revision:     REVISION,
		Reference:    REFERENCE,
		GOVersion:    runtime.Version(),
		BuiltAt:      BUILT,
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
}
