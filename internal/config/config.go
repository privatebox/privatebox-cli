// Package config stores endpoint-scoped sessions in the OS credential store,
// with an owner-only file fallback only when no backend is available.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/privatebox/privatebox-cli/internal/api"
	"github.com/privatebox/privatebox-cli/internal/keyring"
)

const defaultAPIBaseURL = "https://api.privatebox.co.nz/v1"
const keyringService = "privatebox-cli"

// Function boundaries allow failure-path tests without accessing real stores.
var storeGet = keyring.Get
var storeSet = keyring.Set
var storeDelete = keyring.Delete

type Config struct {
	APIBaseURL           string `json:"-"`
	Name                 string `json:"name,omitempty"`
	Email                string `json:"email,omitempty"`
	Token                string `json:"token,omitempty"`
	Storage              string `json:"storage,omitempty"`
	VerificationRequired bool   `json:"verification_required,omitempty"`
	UsingKeyring         bool   `json:"-"`
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".privatebox"), nil
}

func resolveBaseURL() string {
	if v := os.Getenv("PRIVATEBOX_API_URL"); v != "" {
		return v
	}
	return defaultAPIBaseURL
}

func (c *Config) account() string {
	sum := sha256.Sum256([]byte(c.APIBaseURL))
	return "session-" + hex.EncodeToString(sum[:])
}
func (c *Config) path() (string, error) {
	d, err := dir()
	return filepath.Join(d, c.account()+".json"), err
}

func Load() (*Config, error) {
	endpoint, err := api.CanonicalBaseURL(resolveBaseURL())
	if err != nil {
		return nil, err
	}
	cfg := &Config{APIBaseURL: endpoint}
	p, err := cfg.path()
	if err != nil {
		return nil, err
	}
	// Legacy config.json and unscoped keyring slots are never reused: their
	// originating endpoint cannot be established. Users must log in again.
	if info, err := os.Lstat(p); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("session file must be a regular file")
	}
	data, err := os.ReadFile(p)
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("invalid session file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if cfg.Storage == "keyring" {
		cfg.Token = ""
		tok, err := storeGet(keyringService, cfg.account())
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return nil, err
		}
		cfg.Token, cfg.UsingKeyring = tok, true
	} else if cfg.Storage != "file" {
		cfg.Token = ""
	}
	return cfg, nil
}
func (c *Config) GetToken() string { return c.Token }

func (c *Config) SetToken(token string) error {
	if token == "" {
		return fmt.Errorf("API returned an empty session token")
	}
	err := storeSet(keyringService, c.account(), token)
	if err == nil {
		c.Token, c.Storage, c.UsingKeyring = token, "keyring", true
		return c.SaveMeta()
	}
	if !errors.Is(err, keyring.ErrUnavailable) {
		return err
	}
	if c.UsingKeyring || c.Storage == "keyring" {
		return fmt.Errorf("existing keyring session unavailable; restore keyring access before logging in")
	}
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows Credential Manager unavailable; refusing plaintext fallback")
	}
	c.Token, c.Storage, c.UsingKeyring = token, "file", false
	return c.SaveMeta()
}

func (c *Config) ClearToken() error {
	if c.UsingKeyring || c.Storage == "keyring" {
		err := storeDelete(keyringService, c.account())
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return err
		}
	}
	c.Token, c.Name, c.Email, c.Storage = "", "", "", ""
	c.UsingKeyring, c.VerificationRequired = false, false
	return c.SaveMeta()
}

func (c *Config) SaveMeta() error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(d)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("session directory must not be a symlink")
	}
	if err := os.Chmod(d, 0700); err != nil {
		return err
	}
	p, err := c.path()
	if err != nil {
		return err
	}
	if info, err := os.Lstat(p); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("session file must be a regular file")
	}
	copy := *c
	if c.Storage != "file" {
		copy.Token = ""
	}
	data, err := json.MarshalIndent(copy, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(d, ".session-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), p)
}
