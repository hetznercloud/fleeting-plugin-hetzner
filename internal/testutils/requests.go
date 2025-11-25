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
	GetServerTypeCPX22Request = mockutil.Request{
		Method: "GET", Path: "/server_types?name=cpx22",
		Status: 200,
		JSON: schema.ServerTypeListResponse{
			ServerTypes: []schema.ServerType{
				{
					ID:           1,
					Name:         "cpx22",
					Architecture: "x86",
					Locations: []schema.ServerTypeLocation{
						{ID: 1, Name: "fsn1"},
						{ID: 2, Name: "nbg1"},
						{ID: 3, Name: "hel1"},
					},
				},
			},
		},
	}
	GetServerTypeCX23Request = mockutil.Request{
		Method: "GET", Path: "/server_types?name=cx23",
		Status: 200,
		JSON: schema.ServerTypeListResponse{
			ServerTypes: []schema.ServerType{
				{
					ID:           2,
					Name:         "cx23",
					Architecture: "x86",
					Locations: []schema.ServerTypeLocation{
						{ID: 1, Name: "fsn1"},
						{ID: 2, Name: "nbg1"},
						{ID: 3, Name: "hel1"},
					},
				},
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

	GetVolumesRequest = mockutil.Request{
		Method: "GET", Path: "/volumes?label_selector=instance-group%3Dfleeting&page=1",
		Status: 200,
		JSON: schema.VolumeListResponse{
			Volumes: []schema.Volume{},
		},
	}
)
