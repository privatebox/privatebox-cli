//go:build linux || darwin

package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ReadPassword reads a line from stdin with terminal echo disabled,
// so the password isn't printed to the screen.
func ReadPassword() (string, error) {
	disableEcho := exec.Command("stty", "-echo")
	disableEcho.Stdin = os.Stdin
	_ = disableEcho.Run()

	defer func() {
		restoreEcho := exec.Command("stty", "echo")
		restoreEcho.Stdin = os.Stdin
		_ = restoreEcho.Run()
	}()

	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(password, "\r\n"), nil
}
