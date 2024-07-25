package testutils

import (
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func MakeTestClient(endpoint string) *hcloud.Client {
	opts := []hcloud.ClientOption{
		hcloud.WithEndpoint(endpoint),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0), MaxRetries: 3}),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	}

	return hcloud.NewClient(opts...)
}
