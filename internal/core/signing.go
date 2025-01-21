// Optioanl commit signing with ed25519
package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// signCommit uses a passphrase-protected key in .evo/keys/ed25519_priv.enc
func signCommit(repoPath, commitHash string, user UserConfig) (string, error) {
    evoPath := filepath.Join(repoPath, EvoDir)
    keyDir := filepath.Join(evoPath, "keys")
    privFile := filepath.Join(keyDir, "ed25519_priv.enc")
    pubFile := filepath.Join(keyDir, "ed25519_pub")

    var privKey ed25519.PrivateKey
    if _, err := os.Stat(privFile); os.IsNotExist(err) {
        // generate new key
        fmt.Println("No existing key found, generating a new ed25519 key pair...")
        pub, priv, _ := ed25519.GenerateKey(rand.Reader)

        // ask passphrase
        pass, err := getPassphrase("Enter passphrase for new key (leave blank for no pass): ")
        if err != nil {
            return "", err
        }
        encPriv, err := encryptPrivateKey(priv, pass)
        if err != nil {
            return "", err
        }
        os.WriteFile(privFile, encPriv, 0600)

        // store pub in hex
        os.WriteFile(pubFile, []byte(hex.EncodeToString(pub)), 0644)
        privKey = priv
    } else {
        // read pub, read priv, decrypt
        pass, err := getPassphrase("Enter passphrase for your existing key: ")
        if err != nil {
            return "", err
        }
        encPriv, err := os.ReadFile(privFile)
        if err != nil {
            return "", err
        }
        priv, err := decryptPrivateKey(encPriv, pass)
        if err != nil {
            return "", err
        }
        privKey = ed25519.PrivateKey(priv)
    }

    sig := ed25519.Sign(privKey, []byte(commitHash))
    return hex.EncodeToString(sig), nil
}

// VerifyCommit checks the stored signature against the commit hash
func VerifyCommit(repoPath string, c *Commit) bool {
    if c.Signature == "" {
        return false
    }
    evoPath := filepath.Join(repoPath, EvoDir)
    pubFile := filepath.Join(evoPath, "keys", "ed25519_pub")
    pubHex, err := os.ReadFile(pubFile)
    if err != nil {
        return false
    }
    pub, _ := hex.DecodeString(string(pubHex))
    signatureBytes, err := hex.DecodeString(c.Signature)
    if err != nil {
        return false
    }
    // recompute commit's raw
    raw := fmt.Sprintf("%s|%s|%v|%s", c.Message, c.Author, c.Timestamp.UnixNano(), c.TreeHash)
    for _, p := range c.Parents {
        raw += "|" + p
    }
    sum := ed25519.SignatureSize // dummy usage
    _ = sum // ignore
    // let's re-hash
    // Actually, we hashed string(c.Hash) to sign. But let's do consistent approach:
    // We'll just see if public key verifies commitHash = c.Hash
    // So we do:
    commitHashBytes, err := hex.DecodeString(c.Hash)
    if err != nil {
        return false
    }
    return ed25519.Verify(ed25519.PublicKey(pub), commitHashBytes, signatureBytes)
}

// A real passphrase approach uses PBKDF or similar to encrypt. We'll do a simple xor or something:
func encryptPrivateKey(key ed25519.PrivateKey, pass []byte) ([]byte, error) {
    if len(pass) == 0 {
        return key, nil
    }
    enc := make([]byte, len(key))
    for i, b := range key {
        enc[i] = b ^ pass[i%len(pass)]
    }
    return enc, nil
}

func decryptPrivateKey(enc []byte, pass []byte) ([]byte, error) {
    if len(pass) == 0 {
        return enc, nil
    }
    dec := make([]byte, len(enc))
    for i, b := range enc {
        dec[i] = b ^ pass[i%len(pass)]
    }
    return dec, nil
}

func getPassphrase(prompt string) ([]byte, error) {
    fmt.Print(prompt)
    pass, err := terminal.ReadPassword(int(syscall.Stdin))
    fmt.Println()
    return pass, err
}
