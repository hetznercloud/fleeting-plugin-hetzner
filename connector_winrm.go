package aws

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func (g *InstanceGroup) winrm(ctx context.Context, info *provider.ConnectInfo, instance *ec2.Instance) error {
	var out *ec2.GetPasswordDataOutput
	var err error

	for i := 0; i < 120; i++ {
		g.log.Debug("fetching password data", "instance", aws.StringValue(instance.InstanceId), "try", i+1)

		out, err = g.ec2.GetPasswordDataWithContext(ctx, &ec2.GetPasswordDataInput{
			InstanceId: instance.InstanceId,
		})
		if err != nil {
			return fmt.Errorf("fetching password data: %w", err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if aws.StringValue(out.PasswordData) == "" {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	if aws.StringValue(out.PasswordData) == "" {
		return fmt.Errorf("password data empty")
	}

	priv, err := ssh.ParseRawPrivateKey(info.Key)
	if err != nil {
		return fmt.Errorf("reading private key: %w", err)
	}

	decrypter, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("unable to get decrypter from key")
	}

	decodedKey, err := base64.StdEncoding.DecodeString(aws.StringValue(out.PasswordData))
	if err != nil {
		return fmt.Errorf("decoding key: %w", err)
	}

	plain, err := rsa.DecryptPKCS1v15(rand.Reader, decrypter, decodedKey)
	if err != nil {
		return fmt.Errorf("decrypting: %w", err)
	}

	info.Password = string(plain)

	return nil
}
