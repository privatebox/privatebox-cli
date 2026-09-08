package config

import "testing"

func TestResolveBaseURL(t *testing.T) {
	for _, override := range []string{"", "http://127.0.0.1:8080/v1"} {
		t.Run(override, func(t *testing.T) {
			t.Setenv("PRIVATEBOX_API_URL", override)
			want := override
			if want == "" {
				want = "https://api.privatebox.co.nz/v1"
			}
			if got := resolveBaseURL(); got != want {
				t.Fatalf("resolveBaseURL() = %q; want %q", got, want)
			}
		})
	}
}
