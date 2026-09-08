package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// Execute in a subprocess so exit status and stdout are tested as an agent sees them.
func TestCLIHelper(t *testing.T) {
	if os.Getenv("PB_CLI_TEST_HELPER") != "1" {
		return
	}
	for i, v := range os.Args {
		if v == "--" {
			os.Args = append([]string{"privatebox"}, os.Args[i+1:]...)
			break
		}
	}
	Execute()
	os.Exit(0)
}
func TestCLIRegression(t *testing.T) {
	var mu sync.Mutex
	var requests []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		requests = append(requests, body)
		mu.Unlock()
		switch r.URL.Path {
		case "/order/scan":
			w.Write([]byte(`{"status_code":200,"status_message":"Scan queued"}`))
		case "/order/send/cost":
			w.Write([]byte(`{"services":[]}`))
		case "/order/send":
			w.Write([]byte(`{"service":{"id":1,"name":"Fixture"}}`))
		case "/login":
			w.Write([]byte(`{"token":"synthetic-test-session","email":"fixture@example.invalid","verification_required":true}`))
		case "/user/validate_code":
			w.Write([]byte(`{"token":"synthetic-verified-session"}`))
		default:
			w.Write([]byte(`{"items":[{"id":1,"from":{"name":"fixture\u001b[2J\nforged"}}],"current_page":1,"last_page":1,"total":1}`))
		}
	}))
	defer srv.Close()
	home := t.TempDir()
	d := filepath.Join(home, ".privatebox")
	os.MkdirAll(d, 0700)
	sum := sha256.Sum256([]byte(srv.URL))
	session := filepath.Join(d, "session-"+hex.EncodeToString(sum[:])+".json")
	os.WriteFile(session, []byte(`{"storage":"file","token":"synthetic-test-session"}`), 0600)
	exe, _ := os.Executable()
	run := func(input string, args ...string) (string, string, int) {
		t.Helper()
		c := exec.Command(exe, append([]string{"-test.run=^TestCLIHelper$", "--"}, args...)...)
		// No keyring programs on PATH. Linux login tests exercise the file fallback.
		c.Env = append(os.Environ(), "PB_CLI_TEST_HELPER=1", "HOME="+home, "USERPROFILE="+home, "PRIVATEBOX_API_URL="+srv.URL, "PATH="+t.TempDir())
		c.Stdin = strings.NewReader(input)
		var out, stderr bytes.Buffer
		c.Stdout = &out
		c.Stderr = &stderr
		err := c.Run()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		return out.String(), stderr.String(), code
	}
	for _, args := range [][]string{
		{"order", "destroy", "--items", "1", "ignored", "--dry-run"},
		{"order", "destroy", "--items", "1", "--dry-run"},
		{"order", "destroy", "--items", "0", "--yes"},
		{"order", "destroy", "--items", "1"},
		{"order", "scan", "--items", "1", "--destroy"},
		{"order", "scan", "--items", "1,1"},
		{"order", "scan", "--items", "1,0"},
		{"order", "scan", "--items", "1", "--read-only"},
		{"items", "--page", "0"}, {"items", "--page", "1", "--page", "2"},
		{"--no-input=invalid", "status"}, {"status", "--bogus"}, {"auth", "logout", "--bogus"}, {"meta", "countries", "oops"},
		{"auth", "login", "--email", "fixture@example.invalid"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			mu.Lock()
			before := len(requests)
			mu.Unlock()
			out, _, code := run("", append(args, "--json")...)
			var result map[string]any
			if code == 0 || json.Unmarshal([]byte(out), &result) != nil || result["error"] == nil {
				t.Fatal("expected JSON failure", code, out)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(requests) != before {
				t.Fatal("invalid command contacted API")
			}
		})
	}
	out, _, code := run("", "order", "scan", "--items", "1", "--items", "2", "--json")
	if code != 0 || !strings.Contains(out, "Scan queued") || strings.Contains(out, "order_id") {
		t.Fatal(code, out)
	}
	mu.Lock()
	ids := requests[len(requests)-1]["items"].([]any)
	mu.Unlock()
	if len(ids) != 2 {
		t.Fatal("repeated --items lost", ids)
	}
	for _, value := range []string{"0600", "SW1A 1AA"} {
		out, _, code := run("", "order", "send-cost", "--items", "1", "--country", "NZ", "--address", "fixture", "--post-code", value, "--json")
		if code != 0 {
			t.Fatal(out)
		}
		mu.Lock()
		address := requests[len(requests)-1]["address"].(map[string]any)
		mu.Unlock()
		if address["post_code"] != value {
			t.Fatal("postcode changed", address)
		}
	}
	out, _, code = run("", "items")
	if code != 0 || strings.Contains(out, "\x1b") || strings.Contains(out, "\nforged") {
		t.Fatal("unsafe terminal output", out)
	}
	for _, args := range [][]string{{"status"}, {"--version"}, {"--help"}, {"auth", "--help"}, {"order", "send", "--help"}} {
		out, _, code = run("", append(args, "--json")...)
		if code != 0 || !json.Valid([]byte(out)) {
			t.Fatal("not JSON", out)
		}
	}
	out, _, code = run("", "order", "send", "--help")
	if code != 0 || !strings.Contains(out, "post-code") || !strings.Contains(out, "state") {
		t.Fatal("missing subcommand help", out)
	}
	if runtime.GOOS == "linux" {
		out, stderr, code := run("synthetic-password\n", "auth", "login", "--email", "fixture@example.invalid", "--password-stdin", "--json")
		if code != 0 || !json.Valid([]byte(out)) || !strings.Contains(out, "verification_required") {
			t.Fatal(code, out, stderr)
		}
		out, _, code = run("synthetic-code\n", "auth", "verify-code", "--code-stdin", "--json")
		if code != 0 || !strings.Contains(out, "logged_in") {
			t.Fatal(code, out)
		}
		out, _, code = run("", "auth", "logout", "--json")
		if code != 0 {
			t.Fatal(out)
		}
		out, _, _ = run("", "status", "--json")
		if !strings.Contains(out, `"session_saved": false`) {
			t.Fatal("logout retained access", out)
		}
	}
}
