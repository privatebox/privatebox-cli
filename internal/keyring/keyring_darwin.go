//go:build darwin

package keyring

import (
	"bytes"
	"os/exec"
	"strings"
)

// macOS ships the `security` CLI by default, which talks to Keychain.

func available() bool {
	_, err := exec.LookPath("security")
	return err == nil
}

func Set(service, account, secret string) error {
	if !available() {
		return ErrUnavailable
	}
	// -U: update the item in place if it already exists.
	cmd := exec.Command("security", "add-generic-password",
		"-a", account, "-s", service, "-w", secret, "-U")
	return cmd.Run()
}

func Get(service, account string) (string, error) {
	if !available() {
		return "", ErrUnavailable
	}
	var out bytes.Buffer
	cmd := exec.Command("security", "find-generic-password",
		"-a", account, "-s", service, "-w")
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
	cmd := exec.Command("security", "delete-generic-password",
		"-a", account, "-s", service)
	return cmd.Run()
}
