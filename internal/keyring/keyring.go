// Package keyring stores secrets in the OS-native credential store
// when one is available (macOS Keychain, Windows Credential Manager,
// or the Secret Service on Linux via secret-tool). Callers should
// fall back to file storage when ErrUnavailable is returned.
package keyring

import "errors"

var (
	// ErrUnavailable means no OS keyring backend could be used on this
	// machine (e.g. secret-tool isn't installed on this Linux box).
	ErrUnavailable = errors.New("keyring: no OS credential store available")
	// ErrNotFound means the backend is available but has no matching entry.
	ErrNotFound = errors.New("keyring: secret not found")
)
