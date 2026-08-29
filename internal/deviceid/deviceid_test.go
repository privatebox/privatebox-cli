package deviceid

import "testing"

func TestIDStable(t *testing.T) {
	a := ID()
	if a == "" {
		t.Fatal("expected a non-empty device ID on this platform")
	}
	if len(a) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(a))
	}
	if b := ID(); a != b {
		t.Fatal("device ID not stable within process")
	}
}
