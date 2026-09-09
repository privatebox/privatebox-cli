package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadmeDocumentsEndpointScopedSessionFiles(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, `%USERPROFILE%\.privatebox\config.json`) {
		t.Fatal("README still documents the legacy Windows config.json path as current")
	}
	if !strings.Contains(body, `%USERPROFILE%\.privatebox\session-<hash>.json`) {
		t.Fatal("README missing the Windows endpoint-scoped session path")
	}
}

// Execute the documented macOS block with an intentionally wrong checksum.
// Extraction and privileged installation are inert stubs and must never run.
func TestMacOSInstructionsFailClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash documentation test")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	section := strings.SplitN(string(data), "### macOS without Homebrew", 2)[1]
	block := strings.SplitN(strings.SplitN(section, "```bash\n", 2)[1], "\n```", 2)[0]
	d := t.TempDir()
	bin := filepath.Join(d, "bin")
	os.Mkdir(bin, 0700)
	shim := "#!" + bash + "\n" + `set -eu
case "${0##*/}" in
 uname) echo arm64 ;;
 curl)
  output=; latest=false
  while [[ $# -gt 0 ]]; do
   case "$1" in -o) output=$2; shift ;; -w) latest=true; shift ;; esac
   shift
  done
  if $latest; then echo https://github.com/privatebox/privatebox-cli/releases/tag/v9.8.7
  elif [[ $output == checksums.txt ]]; then
   printf '%064d  privatebox_9.8.7_darwin_arm64.tar.gz\n' 0 > "$output"
  else printf 'fixture archive\n' > "$output"; fi ;;
 tar|sudo) echo forbidden >> "$CASE_LOG" ;;
esac
`
	for _, name := range []string{"curl", "uname", "tar", "sudo"} {
		os.WriteFile(filepath.Join(bin, name), []byte(shim), 0700)
	}
	c := exec.Command(bash, "-c", block)
	c.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "CASE_LOG="+filepath.Join(d, "forbidden"))
	out, err := c.CombinedOutput()
	t.Log(string(out))
	if err == nil {
		t.Fatal("checksum failure returned success", string(out))
	}
	if _, err := os.Stat(filepath.Join(d, "forbidden")); !os.IsNotExist(err) {
		b, _ := os.ReadFile(filepath.Join(d, "forbidden"))
		t.Fatal("installation continued after checksum failure", string(b))
	}
}
