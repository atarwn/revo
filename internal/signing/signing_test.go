package signing

import (
	"evo/internal/config"
	"evo/internal/types"
	"os"
	"path/filepath"
	"testing"
)

func TestSigningKeyPair(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "signing_key")

	// Set up config for test
	err := config.SetConfigValue(tmpDir, "signing.keyPath", keyPath)
	if err != nil {
		t.Fatalf("Failed to set config value: %v", err)
	}

	t.Run("Generate_Key_Pair", func(t *testing.T) {
		err := GenerateKeyPair(tmpDir)
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		// Check that key files exist
		if _, err := os.Stat(keyPath); err != nil {
			t.Errorf("Private key file not found: %v", err)
		}
		if _, err := os.Stat(keyPath + ".pub"); err != nil {
			t.Errorf("Public key file not found: %v", err)
		}
	})

	t.Run("Load_Key_Pair", func(t *testing.T) {
		kp, err := LoadKeyPair(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load key pair: %v", err)
		}

		if kp.PrivateKey == nil {
			t.Error("Private key is nil")
		}
		if kp.PublicKey == nil {
			t.Error("Public key is nil")
		}
		if kp.Created.IsZero() {
			t.Error("Creation time not set")
		}
	})

	t.Run("Sign_and_Verify_Commit", func(t *testing.T) {
		commit := &types.Commit{
			Message: "Test commit",
		}

		sig, err := SignCommit(commit, tmpDir)
		if err != nil {
			t.Fatalf("Failed to sign commit: %v", err)
		}
		if sig == "" {
			t.Error("Empty signature returned")
		}

		commit.Signature = sig
		valid, err := VerifyCommit(commit, tmpDir)
		if err != nil {
			t.Errorf("Failed to verify commit: %v", err)
		}
		if !valid {
			t.Error("Signature verification failed")
		}
	})

	t.Run("Invalid_Signature", func(t *testing.T) {
		commit := &types.Commit{
			Message:   "Test commit",
			Signature: "invalid",
		}

		valid, err := VerifyCommit(commit, tmpDir)
		if err == nil {
			t.Error("Expected error for invalid signature")
		}
		if valid {
			t.Error("Invalid signature reported as valid")
		}
	})

	t.Run("Missing_Signature", func(t *testing.T) {
		commit := &types.Commit{
			Message: "Test commit",
		}

		valid, err := VerifyCommit(commit, tmpDir)
		if err == nil {
			t.Error("Expected error for missing signature")
		}
		if valid {
			t.Error("Missing signature reported as valid")
		}
	})
}
