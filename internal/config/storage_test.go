package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/privatebox/privatebox-cli/internal/keyring"
)

func isolatedStore(t *testing.T) map[string]string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	t.Setenv("PRIVATEBOX_API_URL", "")
	get, set, del := storeGet, storeSet, storeDelete
	t.Cleanup(func() { storeGet, storeSet, storeDelete = get, set, del })
	values := map[string]string{}
	storeGet = func(s, a string) (string, error) {
		v, ok := values[a]
		if !ok {
			return "", keyring.ErrNotFound
		}
		return v, nil
	}
	storeSet = func(s, a, v string) error { values[a] = v; return nil }
	storeDelete = func(s, a string) error { delete(values, a); return nil }
	return values
}
func TestSessionIsolationAndLegacy(t *testing.T) {
	values := isolatedStore(t)
	d, _ := dir()
	os.MkdirAll(d, 0700)
	os.WriteFile(filepath.Join(d, "config.json"), []byte(`{"token":"synthetic-legacy"}`), 0600)
	values["session-token"] = "synthetic-legacy"
	c, err := Load()
	if err != nil || c.Token != "" {
		t.Fatal("legacy session reused", err)
	}
	if err = c.SetToken("synthetic-production"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRIVATEBOX_API_URL", "https://testing.example.invalid/v1")
	other, err := Load()
	if err != nil || other.Token != "" {
		t.Fatal("cross-endpoint session reused", err)
	}
	other.SetToken("synthetic-test")
	t.Setenv("PRIVATEBOX_API_URL", "https://API.PRIVATEBOX.CO.NZ:443/v1/")
	same, err := Load()
	if err != nil || same.Token != "synthetic-production" {
		t.Fatal("canonical session not recovered", err)
	}
}
func TestLogoutFailureIsNotSuccess(t *testing.T) {
	isolatedStore(t)
	c, _ := Load()
	if err := c.SetToken("synthetic"); err != nil {
		t.Fatal(err)
	}
	storeDelete = func(s, a string) error { return errors.New("locked") }
	if c.ClearToken() == nil {
		t.Fatal("deletion failure ignored")
	}
	loaded, err := Load()
	if err != nil || loaded.Token != "synthetic" {
		t.Fatal("failed logout lost state", err)
	}
	storeDelete = func(s, a string) error { return keyring.ErrNotFound }
	if err := c.ClearToken(); err != nil {
		t.Fatal(err)
	}
	loaded, err = Load()
	if err != nil || loaded.Token != "" {
		t.Fatal("logout retained session", err)
	}
}
func TestNoFallbackOnDeniedOrLockedStore(t *testing.T) {
	isolatedStore(t)
	c, _ := Load()
	storeSet = func(s, a, v string) error { return errors.New("denied") }
	if c.SetToken("synthetic") == nil {
		t.Fatal("denial should fail")
	}
	p, _ := c.path()
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("denied secret persisted")
	}
}
func TestFallbackPermissionsAndAtomicReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows refuses plaintext fallback")
	}
	isolatedStore(t)
	storeSet = func(s, a, v string) error { return keyring.ErrUnavailable }
	c, _ := Load()
	p, _ := c.path()
	d, _ := dir()
	os.MkdirAll(d, 0755)
	os.Chmod(d, 0755)
	os.WriteFile(p, []byte(`{}`), 0644)
	os.Chmod(p, 0644)
	if err := c.SetToken("synthetic"); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]os.FileMode{d: 0700, p: 0600} {
		info, err := os.Stat(name)
		if err != nil || info.Mode().Perm() != want {
			t.Fatalf("wrong permissions on %s: %v", name, err)
		}
	}
	loaded, err := Load()
	if err != nil || loaded.Token != "synthetic" {
		t.Fatal("fallback not loaded", err)
	}
	// A later keyring session replaces plaintext rather than leaving an old copy.
	storeSet = func(s, a, v string) error { return nil }
	if err := c.SetToken("replacement"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	var saved map[string]any
	json.Unmarshal(data, &saved)
	if _, ok := saved["token"]; ok {
		t.Fatal("plaintext token remained after keyring migration")
	}
	files, _ := filepath.Glob(filepath.Join(d, ".session-*"))
	if len(files) != 0 {
		t.Fatal("temporary session files leaked")
	}
}
func TestSymlinkRefusal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	isolatedStore(t)
	c, _ := Load()
	d, _ := dir()
	os.MkdirAll(d, 0700)
	p, _ := c.path()
	target := filepath.Join(t.TempDir(), "target")
	os.WriteFile(target, []byte("original"), 0600)
	os.Symlink(target, p)
	if c.SaveMeta() == nil {
		t.Fatal("followed a symlink")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "original" {
		t.Fatal("changed symlink target")
	}
}
