package utils

import (
	"crypto"
	"crypto/ed25519"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func encodePublicKey(pub crypto.PublicKey) ([]byte, error) {
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(sshPub), nil
}

func encodePrivateKey(priv crypto.PrivateKey) ([]byte, error) {
	privPem, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(privPem), nil
}

func GenerateSSHKeyPair() ([]byte, []byte, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	pubBytes, err := encodePublicKey(pub)
	if err != nil {
		return nil, nil, err
	}

	privBytes, err := encodePrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	return pubBytes, privBytes, nil
}

type PrivateKeyWithPublic interface {
	crypto.PrivateKey
	Public() crypto.PublicKey
}

func GenerateSSHPublicKey(privBytes []byte) ([]byte, error) {
	priv, err := ssh.ParseRawPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}

	key, ok := priv.(PrivateKeyWithPublic)
	if !ok {
		return nil, fmt.Errorf("key doesn't export PublicKey()")
	}

	pubBytes, err := encodePublicKey(key.Public())
	if err != nil {
		return nil, err
	}

	return pubBytes, nil
}

func GetSSHPublicKeyFingerprint(pubBytes []byte) (string, error) {
	pub, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	if err != nil {
		return "", err
	}

	fingerprint := ssh.FingerprintLegacyMD5(pub)

	return fingerprint, nil
}
