//go:build !windows && !linux && !darwin

package keyring

// Safety net for any other GOOS (e.g. BSDs) — no keyring backend,
// callers fall back to file storage.

func Set(service, account, secret string) error   { return ErrUnavailable }
func Get(service, account string) (string, error) { return "", ErrUnavailable }
func Delete(service, account string) error        { return ErrUnavailable }
