package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func IsTerminal(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }

// ReadLine avoids read-ahead that could consume the next authentication input.
func ReadLine() (string, error) {
	var b strings.Builder
	one := make([]byte, 1)
	for b.Len() < 16384 {
		n, err := os.Stdin.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				return strings.TrimSuffix(b.String(), "\r"), nil
			}
			b.WriteByte(one[0])
		}
		if err != nil {
			if err == io.EOF && b.Len() > 0 {
				return strings.TrimSuffix(b.String(), "\r"), nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("input exceeds 16 KiB")
}

func ReadPassword() (string, error) {
	if !IsTerminal(os.Stdin) {
		return "", fmt.Errorf("interactive terminal required; use --password-stdin with --email for automation")
	}
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(b), err
}
