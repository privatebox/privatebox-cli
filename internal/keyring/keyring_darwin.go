//go:build darwin

package keyring

import (
	"errors"
	native "github.com/zalando/go-keyring"
)

// The maintained macOS backend supplies secret data through a private stdin
// pipe, never process arguments. Backend failures must not trigger file fallback.
func Set(service, account, secret string) error {
	// Bound the interactive security command before the backend starts it.
	if len(secret) > 2048 {
		return errors.New("keyring: session token exceeds supported size")
	}
	if err := translate(native.Set(service, account, secret)); err != nil {
		return err
	}
	saved, err := Get(service, account)
	if err != nil {
		return err
	}
	if saved != secret {
		return errors.New("keyring: token write could not be verified")
	}
	return nil
}
func Get(service, account string) (string, error) {
	value, err := native.Get(service, account)
	return value, translate(err)
}
func Delete(service, account string) error { return translate(native.Delete(service, account)) }
func translate(err error) error {
	if errors.Is(err, native.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return errors.New("keyring: macOS credential operation failed; unlock or allow access to Keychain")
	}
	return nil
}
