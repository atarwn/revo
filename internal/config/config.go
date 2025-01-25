package config

import (
	"encoding/json"
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

// GetConfigValue retrieves a value from the config file
func GetConfigValue(repoPath, key string) (string, error) {
	config, err := loadConfig(repoPath)
	if err != nil {
		return "", err
	}

	value, ok := config[key]
	if !ok {
		return "", fmt.Errorf("no config value for %s", key)
	}

	return value, nil
}

// SetConfigValue stores a value in the config file
func SetConfigValue(repoPath, key, value string) error {
	config, err := loadConfig(repoPath)
	if err != nil {
		// If config doesn't exist, create new map
		config = make(map[string]string)
	}

	config[key] = value

	// Ensure .evo directory exists
	configDir := filepath.Join(repoPath, ".evo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write updated config
	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func loadConfig(repoPath string) (map[string]string, error) {
	configPath := filepath.Join(repoPath, ".evo", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config value for signing.keyPath")
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]string
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
