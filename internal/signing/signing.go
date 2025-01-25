package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"evo/internal/config"
	"evo/internal/types"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Created    time.Time
}

// GenerateKeyPair creates a new Ed25519 key pair and stores it
func GenerateKeyPair(repoPath string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	keyPath, err := getKeyPath(repoPath)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Write private key
	if err := os.WriteFile(keyPath, priv, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key
	pubFile := keyPath + ".pub"
	if err := os.WriteFile(pubFile, pub, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("Generated new Ed25519 key pair:\n")
	fmt.Printf("Private key: %s\n", keyPath)
	fmt.Printf("Public key: %s\n", pubFile)
	return nil
}

// LoadKeyPair loads an existing key pair from disk
func LoadKeyPair(repoPath string) (*KeyPair, error) {
	keyPath, err := getKeyPath(repoPath)
	if err != nil {
		return nil, err
	}

	priv, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Validate private key
	var pk ed25519.PrivateKey
	if len(priv) == ed25519.SeedSize {
		pk = ed25519.NewKeyFromSeed(priv)
	} else if len(priv) == ed25519.PrivateKeySize {
		pk = priv
	} else {
		return nil, fmt.Errorf("invalid Ed25519 key length: %d", len(priv))
	}

	// Load public key
	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d", len(pub))
	}

	return &KeyPair{
		PrivateKey: pk,
		PublicKey:  pub,
		Created:    getFileCreationTime(keyPath),
	}, nil
}

// SignCommit signs a commit using the configured key
func SignCommit(c *types.Commit, repoPath string) (string, error) {
	kp, err := LoadKeyPair(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to load signing key: %w", err)
	}

	msg := types.CommitHashString(c)
	sig := ed25519.Sign(kp.PrivateKey, []byte(msg))
	return hex.EncodeToString(sig), nil
}

// VerifyCommit verifies a commit's signature
func VerifyCommit(c *types.Commit, repoPath string) (bool, error) {
	if c.Signature == "" {
		return false, fmt.Errorf("commit has no signature")
	}

	kp, err := LoadKeyPair(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to load public key: %w", err)
	}

	sigBytes, err := hex.DecodeString(c.Signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature format: %w", err)
	}

	msg := types.CommitHashString(c)
	if !ed25519.Verify(kp.PublicKey, []byte(msg), sigBytes) {
		return false, fmt.Errorf("signature verification failed")
	}

	return true, nil
}

func getKeyPath(repoPath string) (string, error) {
	keyPath, err := config.GetConfigValue(repoPath, "signing.keyPath")
	if err != nil {
		return "", fmt.Errorf("failed to get key path from config: %w", err)
	}
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		keyPath = filepath.Join(home, ".config", "evo", "signing_key")
	}
	return keyPath, nil
}

func getFileCreationTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
