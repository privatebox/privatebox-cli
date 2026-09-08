package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Exercise the actual installer with local downloads and inert package tools.
// No network access, privilege escalation or host package changes are allowed.
func TestInstaller(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux installer requires Bash")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, manager, arch, fault, want string
	}{
		{"debian", "apt-get", "x86_64", "", "privatebox_9.8.7_amd64.deb"},
		{"debian arm", "apt-get", "aarch64", "", "privatebox_9.8.7_arm64.deb"},
		{"rpm", "dnf", "x86_64", "", "privatebox_9.8.7_amd64.rpm"},
		{"rpm arm", "dnf", "arm64", "", "privatebox_9.8.7_arm64.rpm"},
		{"archive", "", "x86_64", "", "privatebox_9.8.7_linux_amd64.tar.gz"},
		{"archive arm", "", "aarch64", "", "privatebox_9.8.7_linux_arm64.tar.gz"},
		{"checksum mismatch", "apt-get", "x86_64", "checksum", ""},
		{"missing checksum", "apt-get", "x86_64", "missing", ""},
		{"duplicate checksum", "apt-get", "x86_64", "duplicate", ""},
		{"download failure", "apt-get", "x86_64", "download", ""},
		{"manifest failure", "apt-get", "x86_64", "manifest", ""},
		{"release failure", "apt-get", "x86_64", "release", ""},
		{"invalid release", "apt-get", "x86_64", "tag", ""},
		{"unsupported architecture", "apt-get", "i686", "", ""},
		{"unsupported OS", "apt-get", "x86_64", "os", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()
			bin := filepath.Join(fixture, "bin")
			if err := os.Mkdir(bin, 0700); err != nil {
				t.Fatal(err)
			}
			write := func(name, content string) {
				t.Helper()
				if err := os.WriteFile(name, []byte(content), 0700); err != nil {
					t.Fatal(err)
				}
			}
			for _, tool := range []string{"awk", "mktemp", "rm", "sha256sum", "cp"} {
				source, err := exec.LookPath(tool)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(source, filepath.Join(bin, tool)); err != nil {
					t.Fatal(err)
				}
			}
			shim := "#!" + bash + "\n" + `set -eu
case "${0##*/}" in
  uname)
    if [[ $1 == -s ]]; then
      if [[ $CASE_FAULT == os ]]; then echo Darwin; else echo Linux; fi
    else echo "$CASE_ARCH"; fi ;;
  curl)
    output=; url=; latest=false
    while [[ $# -gt 0 ]]; do
      case "$1" in
        -o) output=$2; shift ;;
        -w) latest=true; shift ;;
        https://*) url=$1 ;;
      esac
      shift
    done
    if $latest; then
      [[ $CASE_FAULT != release ]] || exit 22
      if [[ $CASE_FAULT == tag ]]; then echo invalid; else echo https://github.com/privatebox/privatebox-cli/releases/tag/v9.8.7; fi
    elif [[ $url == */checksums.txt ]]; then
      [[ $CASE_FAULT != manifest ]] || exit 22
      cp "$CASE_DIR/checksums" "$output"
    else
      [[ $CASE_FAULT != download ]] || exit 22
      echo "$url" >> "$CASE_DIR/downloads"
      echo "${output%/*}" > "$CASE_DIR/download-dir"
      cp "$CASE_DIR/asset" "$output"
    fi ;;
  *) printf '%s %s\n' "${0##*/}" "$*" >> "$CASE_DIR/installs" ;;
esac
`
			for _, tool := range []string{"uname", "curl", "sudo", "tar", "install"} {
				write(filepath.Join(bin, tool), shim)
			}
			if tc.manager != "" {
				write(filepath.Join(bin, tc.manager), shim)
			}
			if tc.manager == "apt-get" {
				write(filepath.Join(bin, "dpkg"), shim)
			}
			write(filepath.Join(fixture, "asset"), "local release fixture")
			digest := fmt.Sprintf("%x", sha256.Sum256([]byte("local release fixture")))
			if tc.fault == "checksum" {
				digest = strings.Repeat("0", 64)
			}
			manifest := ""
			for _, suffix := range []string{"amd64.deb", "arm64.deb", "amd64.rpm", "arm64.rpm", "linux_amd64.tar.gz", "linux_arm64.tar.gz"} {
				manifest += digest + "  privatebox_9.8.7_" + suffix + "\n"
			}
			if tc.fault == "missing" {
				manifest = ""
			}
			if tc.fault == "duplicate" {
				manifest += manifest
			}
			write(filepath.Join(fixture, "checksums"), manifest)
			cmd := exec.Command(bash, "install.sh")
			cmd.Env = append(os.Environ(), "PATH="+bin, "TMPDIR="+fixture,
				"CASE_DIR="+fixture, "CASE_ARCH="+tc.arch, "CASE_FAULT="+tc.fault)
			out, err := cmd.CombinedOutput()
			installs, _ := os.ReadFile(filepath.Join(fixture, "installs"))
			if tc.want == "" {
				if err == nil || len(installs) != 0 {
					t.Fatalf("expected failure without installation: err=%v, installs=%s, output=%s", err, installs, out)
				}
			} else {
				if err != nil {
					t.Fatalf("installer: %v\n%s", err, out)
				}
				downloads, _ := os.ReadFile(filepath.Join(fixture, "downloads"))
				if !strings.Contains(string(downloads), tc.want) || len(installs) == 0 {
					t.Fatalf("wrong asset or no installation: %s / %s", downloads, installs)
				}
			}
			if dir, err := os.ReadFile(filepath.Join(fixture, "download-dir")); err == nil {
				if _, err := os.Stat(strings.TrimSpace(string(dir))); !os.IsNotExist(err) {
					t.Fatal("download directory was not cleaned up")
				}
			}
		})
	}
}
