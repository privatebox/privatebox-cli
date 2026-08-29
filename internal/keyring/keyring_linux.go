//go:build linux

package keyring

import (
	"bytes"
	"os/exec"
	"strings"
)

// Linux has no single standard credential store. secret-tool (part of
// libsecret, commonly preinstalled on GNOME-based desktops) talks to
// the Secret Service D-Bus API. It's absent on many headless/server
// systems, in which case we report ErrUnavailable and the caller
// falls back to file storage.

func available() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func Set(service, account, secret string) error {
	if !available() {
		return ErrUnavailable
	}
	cmd := exec.Command("secret-tool", "store", "--label=privatebox CLI",
		"service", service, "account", account)
	cmd.Stdin = strings.NewReader(secret)
	return cmd.Run()
}

func Get(service, account string) (string, error) {
	if !available() {
		return "", ErrUnavailable
	}
	var out bytes.Buffer
	cmd := exec.Command("secret-tool", "lookup", "service", service, "account", account)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", ErrNotFound
	}
	val := strings.TrimRight(out.String(), "\n")
	if val == "" {
		return "", ErrNotFound
	}
	return val, nil
}

func Delete(service, account string) error {
	if !available() {
		return ErrUnavailable
	}
	cmd := exec.Command("secret-tool", "clear", "service", service, "account", account)
	return cmd.Run()
}
