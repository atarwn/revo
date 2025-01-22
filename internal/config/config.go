package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

// For example: user.name, user.email, signing.keyPath, files.largeThreshold, verifySignatures

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cfgDir := filepath.Join(home, ".config", "evo")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "config.toml"), nil
}

func repoConfigPath(repoPath string) string {
	return filepath.Join(repoPath, ".evo", "config", "config.toml")
}

func loadToml(path string) (*toml.Tree, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		tree, err := toml.TreeFromMap(map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("failed to create empty config: %w", err)
		}
		return tree, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return toml.LoadBytes(b)
}

func saveToml(tree *toml.Tree, path string) error {
	return os.WriteFile(path, []byte(tree.String()), 0644)
}

// SetGlobalConfigValue sets key=val in ~/.config/evo/config.toml
func SetGlobalConfigValue(key, val string) error {
	gp, err := globalConfigPath()
	if err != nil {
		return err
	}
	tree, err := loadToml(gp)
	if err != nil {
		return err
	}
	tree.Set(key, val)
	return saveToml(tree, gp)
}

// SetRepoConfigValue sets key=val in .evo/config/config.toml
func SetRepoConfigValue(repoPath, key, val string) error {
	rp := repoConfigPath(repoPath)
	tree, err := loadToml(rp)
	if err != nil {
		return err
	}
	tree.Set(key, val)
	return saveToml(tree, rp)
}

// GetConfigValue => repo-level override, else global
func GetConfigValue(repoPath, key string) (string, error) {
	// if repoPath is empty or invalid => fallback to global
	var repoTree *toml.Tree
	if repoPath != "" {
		rp := repoConfigPath(repoPath)
		rt, err := loadToml(rp)
		if err == nil {
			repoTree = rt
		}
	}
	globalTree := (*toml.Tree)(nil)
	gp, err := globalConfigPath()
	if err == nil {
		globalTree, _ = loadToml(gp)
	}

	if repoTree != nil {
		if v := repoTree.Get(key); v != nil {
			return fmt.Sprintf("%v", v), nil
		}
	}
	if globalTree != nil {
		if v := globalTree.Get(key); v != nil {
			return fmt.Sprintf("%v", v), nil
		}
	}
	return "", errors.New("no config value for " + key)
}
