package testutils

import (
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

var (
	GetLocationHel1Request = mockutil.Request{
		Method: "GET", Path: "/locations?name=hel1",
		Status: 200,
		JSON: schema.LocationListResponse{
			Locations: []schema.Location{
				{ID: 3, Name: "hel1"},
			},
		},
	}
	GetServerTypeCPX11Request = mockutil.Request{
		Method: "GET", Path: "/server_types?name=cpx11",
		Status: 200,
		JSON: schema.ServerTypeListResponse{
			ServerTypes: []schema.ServerType{
				{ID: 1, Name: "cpx11", Architecture: "x86"},
			},
		},
	}
	GetImageDebian12Request = mockutil.Request{
		Method: "GET", Path: "/images?architecture=x86&include_deprecated=true&name=debian-12",
		Status: 200,
		JSON: schema.ImageListResponse{
			Images: []schema.Image{
				{ID: 114690387, Name: hcloud.Ptr("debian-12"), OSFlavor: "debian", OSVersion: hcloud.Ptr("12"), Architecture: "x86"},
			},
		},
	}
	ListServerEmptyRequest = mockutil.Request{
		Method: "GET", Path: "/servers?label_selector=instance-group%3Dfleeting&page=1",
		Status: 200,
		JSON:   schema.ServerListResponse{},
	}
)
