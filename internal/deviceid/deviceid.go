// Package deviceid provides a stable per-machine identifier derived
// from platform-specific hardware identity.
package deviceid

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	once     sync.Once
	cachedID string
)

// ID returns a SHA-256 derived device ID for the current machine. The
// same machine always yields the same ID across logins and restarts.
// It returns an empty string if no identifier could be read.
func ID() string {
	once.Do(func() {
		raw, err := machineID()
		if err != nil {
			return
		}
		sum := sha256.Sum256([]byte(raw))
		cachedID = hex.EncodeToString(sum[:])
	})
	return cachedID
}

func machineID() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return readFirstExisting("/etc/machine-id", "/var/lib/dbus/machine-id")
	case "darwin":
		return darwinID()
	case "windows":
		return windowsID()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func readFirstExisting(paths ...string) (string, error) {
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err == nil {
			if id := strings.TrimSpace(string(b)); id != "" {
				return id, nil
			}
		}
	}
	return "", errors.New("no machine id file found")
}

var platformUUIDRe = regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)

func darwinID() (string, error) {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", err
	}
	m := platformUUIDRe.FindSubmatch(out)
	if len(m) < 2 {
		return "", errors.New("IOPlatformUUID not found")
	}
	return string(m[1]), nil
}

func windowsID() (string, error) {
	out, err := exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid").Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "", errors.New("MachineGuid not found")
	}
	return fields[len(fields)-1], nil
}
