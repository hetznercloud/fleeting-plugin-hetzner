package hetzner

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/utils"
)

func (g *InstanceGroup) UploadSSHPublicKey(ctx context.Context, pub []byte) (sshKey *hcloud.SSHKey, err error) {
	fingerprint, err := utils.GetSSHPublicKeyFingerprint(pub)
	if err != nil {
		return nil, fmt.Errorf("could not get ssh key fingerprint: %w", err)
	}

	sshKey, _, err = g.client.SSHKey.GetByFingerprint(ctx, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("could not get ssh key: %w", err)
	}
	if sshKey != nil {
		g.log.Info("using existing ssh key", "name", sshKey.Name, "fingerprint", sshKey.Fingerprint)
		return sshKey, nil
	}

	sshKey, _, err = g.client.SSHKey.GetByName(ctx, g.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get ssh key: %w", err)
	}
	if sshKey != nil {
		g.log.Warn("deleting existing ssh key", "name", sshKey.Name, "fingerprint", sshKey.Fingerprint)
		_, err = g.client.SSHKey.Delete(ctx, sshKey)
		if err != nil {
			return nil, fmt.Errorf("could not delete ssh key: %w", err)
		}
	}

	g.log.Info("uploading ssh key", "name", g.Name, "fingerprint", fingerprint)
	sshKey, _, err = g.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      g.Name,
		Labels:    g.labels,
		PublicKey: string(pub),
	})
	if err != nil {
		return nil, fmt.Errorf("could not upload ssh key: %w", err)
	}

	return sshKey, nil
}
