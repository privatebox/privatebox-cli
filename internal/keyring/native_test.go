package keyring

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

// Opt-in integration test uses a unique disposable item, never a customer session.
func TestNativeStoreRoundTrip(t *testing.T) {
	if os.Getenv("PRIVATEBOX_TEST_NATIVE_KEYRING") != "1" {
		t.Skip("native store test is opt-in")
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	service, account := "privatebox-cli-regression", hex.EncodeToString(id)
	t.Cleanup(func() {
		if err := Delete(service, account); err != nil && !errors.Is(err, ErrNotFound) {
			t.Error("could not remove disposable keyring fixture", err)
		}
	})
	if err := Set(service, account, "synthetic-fixture-one"); err != nil {
		t.Fatal("native set failed", err)
	}
	got, err := Get(service, account)
	if err != nil || got != "synthetic-fixture-one" {
		t.Fatal("native readback failed", err)
	}
	if err := Set(service, account, "synthetic-fixture-two"); err != nil {
		t.Fatal("native update failed", err)
	}
	got, err = Get(service, account)
	if err != nil || got != "synthetic-fixture-two" {
		t.Fatal("native update readback failed", err)
	}
	if err := Delete(service, account); err != nil {
		t.Fatal("native delete failed", err)
	}
	if _, err := Get(service, account); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted item still accessible", err)
	}
}
