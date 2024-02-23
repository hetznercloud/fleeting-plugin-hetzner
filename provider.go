package hetzner

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/hetzner"
	"golang.org/x/crypto/ssh"
	"path"
	"strconv"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

var newClient = hetzner.New

var sshPrivateKeys = make(map[string][]byte)

type InstanceGroup struct {
	AccessToken           string   `json:"access_token"`
	Location              string   `json:"location"`
	ServerType            string   `json:"server_type"`
	Image                 string   `json:"image"`
	DisablePublicNetworks []string `json:"disable_public_networks"`
	PrivateNetworks       []string `json:"private_networks"`

	// Because of limitations in the Hetzner API, instance groups do not formally exist in the
	// Hetzner API. The Name here is mapped to a label which is set on all machines created in this
	// "instance group".
	Name string `json:"name"`

	log               hclog.Logger
	client            hetzner.Client
	size              int
	enablePublicIPv4  bool
	enablePublicIPv6  bool
	privateNetworkIDs []int

	settings provider.Settings
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (provider.ProviderInfo, error) {
	cfg := hetzner.Config{
		AccessToken: g.AccessToken,
		Location:    g.Location,
		ServerType:  g.ServerType,
		Image:       g.Image,
	}

	if cfg.AccessToken == "" {
		return provider.ProviderInfo{}, fmt.Errorf("the plugin_config must contain an access_token setting, containing a valid Hetzner Cloud API token")
	}

	if cfg.Location == "" {
		return provider.ProviderInfo{}, fmt.Errorf("the plugin_config must contain a location setting, which is set to a Hetzner Cloud location: https://docs.hetzner.com/cloud/general/locations/")
	}

	if cfg.ServerType == "" {
		return provider.ProviderInfo{}, fmt.Errorf("the plugin_config must contain a server_type setting, which is set to a Hetzner Cloud server type: https://docs.hetzner.com/cloud/servers/overview/")
	}

	if cfg.Image == "" {
		return provider.ProviderInfo{}, fmt.Errorf("the plugin_config must contain a image setting, which is set to a Hetzner Cloud image. If you have the hcloud CLI installed, you can list available images using `hcloud image list --type system`")
	}

	if g.Name == "" {
		return provider.ProviderInfo{}, fmt.Errorf("the plugin_config must contain a name setting, which is the desired \"instance group\" for the runner. This is used as a prefix for the server names, among other things")
	}

	// Enable both of these by default, unless otherwise specified in the config file
	g.enablePublicIPv4 = true
	g.enablePublicIPv6 = true

	for _, str := range g.DisablePublicNetworks {
		if str == "ipv4" {
			g.enablePublicIPv4 = false
		} else if str == "ipv6" {
			g.enablePublicIPv6 = false
		} else {
			return provider.ProviderInfo{}, fmt.Errorf("unexpected value found in disable_public_networks setting: '%v'. Only 'ipv4' and 'ipv6' are supported", str)
		}
	}

	var err error

	g.client, err = newClient(cfg, Version.Name, Version.String())

	if err != nil {
		return provider.ProviderInfo{}, fmt.Errorf("creating Hetzner client failed: %w", err)
	}

	g.log = log.With("location", cfg.Location, "name", g.Name)
	g.settings = settings

	if _, err := g.getServersInGroup(ctx); err != nil {
		return provider.ProviderInfo{}, err
	}

	// Resolve network names to IDs once, at startup. Note: this means that adding/removing networks
	// while the plugin is running will not work. In our experience, networks are not recreated that
	// frequently, but if this causes problems for you, feel free to raise an issue about it.
	for _, networkName := range g.PrivateNetworks {
		network, err := g.client.GetNetwork(ctx, networkName)

		if err != nil {
			return provider.ProviderInfo{}, fmt.Errorf("retrieving network failed: %w", err)
		}

		if network == nil {
			return provider.ProviderInfo{}, fmt.Errorf("network '%v' not found", networkName)
		}

		g.privateNetworkIDs = append(g.privateNetworkIDs, network.ID)
	}

	return provider.ProviderInfo{
		ID:        path.Join("hetzner", cfg.Location, g.Name),
		MaxSize:   1000,
		Version:   Version.String(),
		BuildInfo: Version.BuildInfo(),
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(id string, state provider.State)) error {
	instances, err := g.getServersInGroup(ctx)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		state := provider.StateCreating

		switch instance.Status {
		case hcloud.ServerStatusStopping, hcloud.ServerStatusDeleting:
			state = provider.StateDeleting

		// Servers always go through `initializing` and `off` when they are created. Since this plugin never
                // "shuts servers down" to power them on later, we are quite safe to assume that "off" here means
                // that the server is still in the initialization phase.
		case hcloud.ServerStatusOff:
			state = provider.StateCreating

		case hcloud.ServerStatusInitializing, hcloud.ServerStatusStarting:
			state = provider.StateCreating

		case hcloud.ServerStatusRunning:
			state = provider.StateRunning

		// TODO: The following are currently not handled. Should be safe, since our plugin should
		// TODO: never cause any of the servers created using it to have any of these states.
		// hcloud.ServerStatusMigrating
		// hcloud.ServerStatusRebuilding
		// hcloud.ServerStatusUnknown
		default:
			return fmt.Errorf("unexpected instance status encountered: %v", instance.Status)
		}

		update(strconv.Itoa(instance.ID), state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	for i := 0; i < delta; i++ {
		var b [8]byte
		_, err := rand.Read(b[:])

		if err != nil {
			return 0, fmt.Errorf("creating random instance name: %w", err)
		}

		serverName := g.Name + "-" + hex.EncodeToString(b[:])

		sshPublicKey, sshPrivateKey, err := createSshKeyPair()

		if err != nil {
			return i + 1, fmt.Errorf("error creating SSH key for server: %w", err)
		}

		sshPrivateKeys[serverName] = sshPrivateKey

		_, err = g.client.CreateServer(ctx, serverName, g.Name, sshPublicKey, g.enablePublicIPv4, g.enablePublicIPv6, g.privateNetworkIDs)

		if err != nil {
			return i + 1, fmt.Errorf("error creating server: %w", err)
		}
	}

	g.size += delta

	return delta, nil
}

func createSshKeyPair() (string, []byte, error) {
	// Generate a new private/public keypair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)

	if err != nil {
		return "", nil, fmt.Errorf("generating private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)

	// Convert the public key to ssh authorized_keys format
	pub, err := ssh.NewPublicKey(privateKey.Public())

	if err != nil {
		return "", nil, err
	}

	var publicKey = string(ssh.MarshalAuthorizedKey(pub))

	return publicKey, privateKeyPEM, nil
}

func (g *InstanceGroup) Decrease(ctx context.Context, instances []string) ([]string, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	var succeeded []string

	for _, instance := range instances {
		server, err := g.client.GetServer(ctx, instance)

		if err != nil {
			return succeeded, err
		}

		// TODO: The lack of "transactions" here is a bit unpleasant. We may end up deleting the
		// TODO: server but not the SSH key, in case one of the API requests for the latter
		// TODO: operation fails.
		err = g.client.DeleteServer(ctx, instance)

		if err != nil {
			g.log.Warn("Deleting server failed", "instance", instance, "err", err)
			return succeeded, err
		}

		sshKey, err := g.client.GetSSHKeyByName(ctx, server.Name)

		if err != nil {
			return succeeded, err
		}

		if sshKey == nil {
			g.log.Warn("SSH key unexpectedly not found", "name", server.Name)
		} else {
			err = g.client.DeleteSSHKey(ctx, sshKey.ID)
		}

		if err != nil {
			return succeeded, err
		}

		g.size--

		succeeded = append(succeeded, instance)
	}

	return succeeded, nil
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, id string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	if info.Protocol == provider.ProtocolWinRM {
		return info, fmt.Errorf("plugin does not support the WinRM protocol")
	}

	if info.Key != nil {
		return info, fmt.Errorf("plugin does not support providing an SSH key in advance")
	}

	server, err := g.client.GetServer(ctx, id)

	if err != nil {
		return info, fmt.Errorf("error getting server: %w", err)
	}

	if server == nil {
		return info, fmt.Errorf("fetching instance %v: not found", id)
	}

	info.ID = id
	info.OS = server.Image.OSFlavor

	// TODO: get this from server-type API
	// info.Arch - Hetzner provides a "server type" API (https://docs.hetzner.cloud/#server-types), but regretfully the architecture field contains "

	//instance := output.Reservations[0].Instances[0]
	//
	//if info.OS == "" {
	//	switch {
	//	case instance.Architecture == types.ArchitectureValuesX8664Mac ||
	//		instance.Architecture == types.ArchitectureValuesArm64Mac:
	//		info.OS = "darwin"
	//	case strings.EqualFold(string(instance.Platform), string(types.PlatformValuesWindows)):
	//		info.OS = "windows"
	//	default:
	//		info.OS = "linux"
	//	}
	//}
	//
	//if info.Arch == "" {
	//	switch instance.Architecture {
	//	case types.ArchitectureValuesI386:
	//		info.Arch = "386"
	//	case types.ArchitectureValuesX8664, types.ArchitectureValuesX8664Mac:
	//		info.Arch = "amd64"
	//	case types.ArchitectureValuesArm64, types.ArchitectureValuesArm64Mac:
	//		info.Arch = "arm64"
	//	}
	//}
	//

	if info.Username == "" {
		info.Username = "root"
	}

	info.Key = sshPrivateKeys[server.Name]

	if info.Key == nil {
		g.log.Error("Key not found", "instance", id, "server_name", server.Name)
	}

	if len(server.PrivateNet) >= 1 {
		info.InternalAddr = server.PrivateNet[0].IP.String()
	}

	info.ExternalAddr = server.PublicNet.IPv4.IP.String()

	if info.Protocol == "" {
		info.Protocol = provider.ProtocolSSH
	}

	return info, err
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	// If Decrease() has been called consistently, there shouldn't be any SSH keys left to delete
	// here. Still, it feels like good practice to at least check if there are anyone left.
	sshKeys, err := g.client.GetSSHKeysInInstanceGroup(ctx, g.Name)

	if err != nil {
		return err
	}

	for _, sshKey := range sshKeys {
		innerErr := g.client.DeleteSSHKey(ctx, sshKey.ID)

		// Just store it for now, but keep deleting keys. It's better to delete 9 out of 10 keys if
		// e.g. deleting the fifth key failed.
		if innerErr != nil {
			g.log.Warn("Error deleting SSH key", "err", innerErr)
			err = innerErr
		}
	}

	// Likewise with server instances; check that we don't leave any stray ones hanging around,
	// which would waste €€€ since they would essentially never be terminated. (Well, this is
	// technically not true since the fleeting mechanism seems to detect "no data on pre-existing
	// instance so removing for safety". But it's still a good idea to remove these servers here
	// since we have no guarantee as to when the plugin will run the next time)
	servers, laterErr := g.client.GetServersInInstanceGroup(ctx, g.Name)

	if laterErr == nil {
		for _, server := range servers {
			innerErr := g.client.DeleteServer(ctx, strconv.Itoa(server.ID))

			// As with SSH keys, save it for now and keep deleting servers.
			if innerErr != nil {
				g.log.Warn("Error deleting server", "err", innerErr)
				err = innerErr
			}
		}
	} else {
		err = laterErr
	}

	return err
}

func (g *InstanceGroup) getServersInGroup(ctx context.Context) ([]*hcloud.Server, error) {
	servers, err := g.client.GetServersInInstanceGroup(ctx, g.Name)

	if err != nil {
		return nil, fmt.Errorf("GetServersInInstanceGroup: %w", err)
	}

	// Workaround for tests which may have added/removed servers without calling our Increase() or
	// Decrease() methods.
	g.size = len(servers)

	return servers, nil
}
