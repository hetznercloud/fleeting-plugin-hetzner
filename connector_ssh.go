package hetzner

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

type PrivPub interface {
	crypto.PrivateKey
	Public() crypto.PublicKey
}

func (g *InstanceGroup) ssh(ctx context.Context, info *provider.ConnectInfo, instance types.Instance) error {
	var key PrivPub
	var err error

	if info.Key != nil {
		priv, err := ssh.ParseRawPrivateKey(info.Key)
		if err != nil {
			return fmt.Errorf("reading private key: %w", err)
		}
		var ok bool
		key, ok = priv.(PrivPub)
		if !ok {
			return fmt.Errorf("key doesn't export PublicKey()")
		}
	} else {
		key, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return fmt.Errorf("generating private key: %w", err)
		}

		info.Key = pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey)),
			},
		)
	}

	sshPubKey, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return fmt.Errorf("generating ssh public key: %w", err)
	}

	result, err := g.client.SendSSHPublicKey(ctx, &ec2instanceconnect.SendSSHPublicKeyInput{
		AvailabilityZone: instance.Placement.AvailabilityZone,
		InstanceId:       instance.InstanceId,
		InstanceOSUser:   &info.Username,
		SSHPublicKey:     aws.String(string(ssh.MarshalAuthorizedKey(sshPubKey))),
	})
	if err != nil {
		return fmt.Errorf("sending ssh key: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("sending ssh key: operation failed")
	}

	expires := time.Now().Add(60 * time.Second)
	info.Expires = &expires

	return nil
}
