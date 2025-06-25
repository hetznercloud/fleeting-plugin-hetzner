package instancegroup

type Config struct {
	// Location is the Hetzner Cloud "Location" (name or id) to create the server in.
	// Run `hcloud location list` to list available locations.
	Location string

	// ServerTypes is a list of Hetzner Cloud "Server Type" (name or id) to create the server
	// with. Run `hcloud server-type list` to list available server types.
	ServerTypes []string

	// Image is the Hetzner Cloud "Image" (name or id) to create the server with. Run
	// `hcloud image list` to list available images.
	Image string

	// UserData is the data available to initialization framework that may run after the
	// server boot.
	UserData string

	// SSHKeys is a list of Hetzner Cloud "SSH Key" (name or id) to create the server
	// with. Run `hcloud ssh-key list` to list available ssh-keys.
	SSHKeys []string

	// PublicIPv4Disabled disables the server public IPv4.
	PublicIPv4Disabled bool
	// PublicIPv6Disabled disables the server public IPv6.
	PublicIPv6Disabled bool
	// PublicIPPoolEnabled enables the public IP pool, which offers a way to have
	// predictable public IPs attached to new servers during there creations.
	PublicIPPoolEnabled bool
	// PublicIPPoolSelector is a label selector (https://docs.hetzner.cloud/reference/cloud#label-selector)
	// used to filter the IPs when populating the IP pool.
	PublicIPPoolSelector string

	// PrivateNetworks is a list of Hetzner Cloud "Network" (name or id) to attach to
	// the server. Run `hcloud network list` to list available ssh-keys.
	PrivateNetworks []string

	// VolumeSize is the size in GB of the volume that will be attached to the server.
	VolumeSize int

	// Labels is a map of key value pairs to create the server with.
	Labels map[string]string
}
