package instancegroup

import "github.com/hetznercloud/hcloud-go/v2/hcloud"

type resourceActions struct {
	ID      int64
	Actions []*hcloud.Action
}

func newResourceActions(id int64, actions ...*hcloud.Action) resourceActions {
	return resourceActions{ID: id, Actions: actions}
}
