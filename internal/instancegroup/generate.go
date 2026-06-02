package instancegroup

//go:generate go run -modfile=../../tools/go.mod go.uber.org/mock/mockgen -package instancegroup -destination zz_mock_instancegroup.go gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup InstanceGroup
