//go:build linux

package keyring

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretServiceFailureClassification(t *testing.T) {
	d := t.TempDir()
	t.Setenv("PATH", d)
	if _, err := Get("fixture", "fixture"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	script := filepath.Join(d, "secret-tool")
	os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0700)
	if _, err := Get("fixture", "fixture"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := Delete("fixture", "fixture"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	os.WriteFile(script, []byte("#!/bin/sh\necho 'locked' >&2\nexit 1\n"), 0700)
	if _, err := Get("fixture", "fixture"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatal("locked store treated as absent", err)
	}
	if err := Delete("fixture", "fixture"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatal("locked deletion treated as absent", err)
	}
}
func TestSecretNeverPassedInArguments(t *testing.T) {
	d := t.TempDir()
	t.Setenv("PATH", d)
	script := filepath.Join(d, "secret-tool")
	// A synthetic fixture only; no real credentials or store access.
	body := "#!/bin/sh\nfor arg in \"$@\"; do case \"$arg\" in *synthetic*) exit 2 ;; esac; done\nIFS= read -r input || :\n[ \"$input\" = synthetic-fixture ]\n"
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Set("fixture", "fixture", strings.Join([]string{"synthetic", "fixture"}, "-")); err != nil {
		t.Fatal(err)
	}
}
