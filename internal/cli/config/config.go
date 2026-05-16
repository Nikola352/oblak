package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Profile struct {
	AuthID    string `toml:"auth_id"`
	SecretKey string `toml:"secret_key"`
	Endpoint  string `toml:"endpoint"`
}

type Config map[string]Profile

func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".oblak", "config.toml")
}

func Load() (Config, error) {
	cfg := make(Config)
	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, toml.Unmarshal(data, &cfg)
}

func (c Config) Save() error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func ActiveProfileName() string {
	if name := os.Getenv("OBLAK_PROFILE"); name != "" {
		return name
	}
	return "default"
}

func (c Config) ActiveProfile() (Profile, error) {
	name := ActiveProfileName()
	p, ok := c[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found: run `oblak configure`", name)
	}
	return p, nil
}
