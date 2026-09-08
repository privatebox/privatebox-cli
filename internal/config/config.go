// Package config persists local CLI state:
// logged-in email, and the session token. The token is stored in the
// OS keyring when one is available, and only falls back to the config
// file (0600) otherwise.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/privatebox/privatebox-cli/internal/keyring"
)

const (
	// Tests can override the production endpoint with PRIVATEBOX_API_URL.
	defaultAPIBaseURL = "https://api.privatebox.co.nz/v1"

	keyringService = "privatebox-cli"
	keyringAccount = "session-token"
)

type Config struct {
	APIBaseURL string `json:"-"`
	Name       string `json:"name,omitempty"`
	Email      string `json:"email,omitempty"`

	// Token is only written to disk when no OS keyring backend is
	// available; otherwise it's kept out of the file entirely.
	Token string `json:"token,omitempty"`

	UsingKeyring bool `json:"-"`
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".privatebox"), nil
}

func path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

func resolveBaseURL() string {
	if v := os.Getenv("PRIVATEBOX_API_URL"); v != "" {
		return v
	}
	return defaultAPIBaseURL
}

// Load reads local config from disk, then checks the OS keyring for a
// session token, which takes priority over any fallback token in the file.
func Load() (*Config, error) {
	cfg := &Config{APIBaseURL: resolveBaseURL()}

	p, err := path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err == nil {
		if uerr := json.Unmarshal(data, cfg); uerr != nil {
			return nil, uerr
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	cfg.APIBaseURL = resolveBaseURL()

	if tok, kerr := keyring.Get(keyringService, keyringAccount); kerr == nil {
		cfg.Token = tok
		cfg.UsingKeyring = true
	}

	return cfg, nil
}

// GetToken returns the current session token, wherever it's stored.
func (c *Config) GetToken() string {
	return c.Token
}

// SetToken stores a new session token: OS keyring first, config file
// (owner-only permissions) as the fallback.
func (c *Config) SetToken(token string) error {
	if err := keyring.Set(keyringService, keyringAccount, token); err == nil {
		c.Token = token
		c.UsingKeyring = true
		return c.saveMeta(false) // token lives in the keyring, not the file
	}
	c.Token = token
	c.UsingKeyring = false
	return c.saveMeta(true)
}

// ClearToken removes the session token from wherever it's stored.
func (c *Config) ClearToken() error {
	_ = keyring.Delete(keyringService, keyringAccount) // best-effort
	c.Token = ""
	c.UsingKeyring = false
	return c.saveMeta(true)
}

// SaveMeta persists non-secret fields (email, pending challenge ID,
// API URL) without changing how the token is stored.
func (c *Config) SaveMeta() error {
	return c.saveMeta(!c.UsingKeyring)
}

func (c *Config) saveMeta(includeToken bool) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}

	p, err := path()
	if err != nil {
		return err
	}

	toWrite := *c
	if !includeToken {
		toWrite.Token = ""
	}

	data, err := json.MarshalIndent(toWrite, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}
