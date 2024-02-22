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
	"golang.org/x/crypto/ssh"
	"path"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"gitlab.com/hiboxsystems/fleeting-plugin-hetzner/internal/hetzner"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

var newClient = hetzner.New

var sshPrivateKeys = make(map[string][]byte)

type InstanceGroup struct {
	AccessToken string `json:"access_token"`
	Location    string `json:"location"`
	ServerType  string `json:"server_type"`
	Image       string `json:"image"`

	// Because of limitations in the Hetzner API, instance groups do not formally exist in the
	// Hetzner API. The Name here is mapped to a label which is set on all machines created in this
	// "instance group".
	Name string `json:"name"`

	log    hclog.Logger
	client hetzner.Client
	size   int

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

		// Workaround for potential bug in the Hetzner API:
		// https://gitlab.com/fleeting-plugin-hetzner/fleeting-plugin-hetzner/-/issues/1#note_1781403906
		// Hetzner support ticket #2024022103016195
		case hcloud.ServerStatusOff:
			state = provider.StateCreating

		case hcloud.ServerStatusInitializing, hcloud.ServerStatusStarting:
			state = provider.StateCreating

		case hcloud.ServerStatusRunning:
			state = provider.StateRunning

		// TODO: how about these? What should we map them to in the Fleeting world view?
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

		// Save the PEM key in our map, but in byte[] format since that's what ConnectInfo will
		// return.
		sshPrivateKeys[serverName] = []byte(sshPrivateKey)

		_, err = g.client.CreateServer(ctx, serverName, g.Name, sshPublicKey)

		if err != nil {
			return i + 1, fmt.Errorf("error creating server: %w", err)
		}
	}

	g.size += delta

	return delta, nil
}

func createSshKeyPair() (string, string, error) {
	// Implementation based on example from https://stackoverflow.com/a/64178933/227779, by Anders
	// Pitman and Greg (CC BY-SA 4.0)

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)

	if err != nil {
		return "", "", err
	}

	// Write private key as PEM
	var privKeyBuf strings.Builder

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}

	// Generate and write public key in authorized_keys format
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	return pubKeyBuf.String(), privKeyBuf.String(), nil
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
