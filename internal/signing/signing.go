package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"evo/internal/config"
	"evo/internal/types"
	"fmt"
	"os"
)

// GenerateKeyPair is optional to create a key if the user calls it
func GenerateKeyPair(repoPath string) error {
	priv, pub, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	keyPath, _ := config.GetConfigValue(repoPath, "signing.keyPath")
	if keyPath == "" {
		keyPath = defaultKeyPath()
	}
	if err := os.WriteFile(keyPath, priv, 0600); err != nil {
		return err
	}
	fmt.Println("Wrote private key to", keyPath)
	pubFile := keyPath + ".pub"
	if err := os.WriteFile(pubFile, pub, 0644); err != nil {
		return err
	}
	fmt.Println("Wrote public key to", pubFile)
	return nil
}

func defaultKeyPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.evo_signing_key"
}

func SignCommit(c *types.Commit) (string, error) {
	keyPath, _ := config.GetConfigValue("", "signing.keyPath")
	if keyPath == "" {
		keyPath = defaultKeyPath()
	}
	priv, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("cannot read signing key: %w", err)
	}
	var pk ed25519.PrivateKey
	if len(priv) == ed25519.SeedSize {
		pk = ed25519.NewKeyFromSeed(priv)
	} else if len(priv) == ed25519.PrivateKeySize {
		pk = priv
	} else {
		return "", fmt.Errorf("invalid ed25519 key length: %d", len(priv))
	}
	msg := types.CommitHashString(c)
	sig := ed25519.Sign(pk, []byte(msg))
	return hex.EncodeToString(sig), nil
}

func VerifyCommit(c *types.Commit) bool {
	if c.Signature == "" {
		return false
	}
	keyPath, _ := config.GetConfigValue("", "signing.keyPath")
	if keyPath == "" {
		keyPath = defaultKeyPath()
	}
	pubFile := keyPath + ".pub"
	pub, err := os.ReadFile(pubFile)
	if err != nil {
		return false
	}
	if len(pub) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(c.Signature)
	if err != nil {
		return false
	}
	msg := types.CommitHashString(c)
	return ed25519.Verify(pub, []byte(msg), sigBytes)
}
