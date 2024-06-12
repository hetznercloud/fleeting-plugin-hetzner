package instancegroup

type Config struct {
	// Location is the Hetzner Cloud "Location" (name or id) to create the server in.
	// Run `hcloud location list` to list available locations.
	Location string

	// ServerType is the Hetzner Cloud "Server Type" (name or id) to create the server
	// with. Run `hcloud server-type list` to list available server types.
	ServerType string

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

	// PrivateNetworks is a list of Hetzner Cloud "Network" (name or id) to attach to
	// the server. Run `hcloud network list` to list available ssh-keys.
	PrivateNetworks []string

	// Labels is a map of key value pairs to create the server with.
	Labels map[string]string
}
