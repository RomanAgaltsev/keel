// Package config manages keel's own user configuration (default author,
// provider). Tokens are never persisted - they come from the environment.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is keel's user-level configuration.
type Config struct {
	AuthorName  string `yaml:"author_name,omitempty"`
	AuthorEmail string `yaml:"author_email,omitempty"`
	Provider    string `yaml:"provider,omitempty"`
}

// Path returns the default config path ($UserConfigDir/keel/config.yaml).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, "keel", "config.yaml"), nil
}

// LoadFrom reads config from path. A missing file yields a zero Config, no error.
func LoadFrom(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return c, nil
}

// SaveTo writes config to path, creating parent dirs.
func SaveTo(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
