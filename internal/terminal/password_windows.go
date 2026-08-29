//go:build windows

package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const enableEchoInput = 0x0004

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

// ReadPassword reads a line from stdin with terminal echo disabled,
// so the password isn't printed to the screen. Uses the Windows
// console API directly (stdlib syscall package only, no external deps).
func ReadPassword() (string, error) {
	handle := syscall.Handle(os.Stdin.Fd())

	var oldMode uint32
	_, _, _ = procGetConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&oldMode)))

	newMode := oldMode &^ enableEchoInput
	_, _, _ = procSetConsoleMode.Call(uintptr(handle), uintptr(newMode))

	defer func() {
		_, _, _ = procSetConsoleMode.Call(uintptr(handle), uintptr(oldMode))
	}()

	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(password, "\r\n"), nil
}
