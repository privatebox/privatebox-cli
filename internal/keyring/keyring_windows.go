//go:build windows

package keyring

import (
	"syscall"
	"unsafe"
)

// Windows Credential Manager, via the advapi32.dll Cred* APIs.
// Uses only the stdlib syscall package — no external dependencies.

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

func target(service, account string) string {
	return service + ":" + account
}

func Set(service, account, secret string) error {
	targetName, err := syscall.UTF16PtrFromString(target(service, account))
	if err != nil {
		return err
	}
	userName, err := syscall.UTF16PtrFromString(account)
	if err != nil {
		return err
	}
	blob := []byte(secret)
	if len(blob) == 0 {
		blob = []byte{0}
	}

	cred := credential{
		Type:               credTypeGeneric,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(secret)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocalMachine,
		UserName:           userName,
	}

	ret, _, err := procCredWrite.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if ret == 0 {
		return err
	}
	return nil
}

func Get(service, account string) (string, error) {
	targetName, err := syscall.UTF16PtrFromString(target(service, account))
	if err != nil {
		return "", err
	}

	var pcred *credential
	ret, _, _ := procCredRead.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&pcred)),
	)
	if ret == 0 {
		return "", ErrNotFound
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(pcred)))

	blob := unsafe.Slice(pcred.CredentialBlob, pcred.CredentialBlobSize)
	return string(blob), nil
}

func Delete(service, account string) error {
	targetName, err := syscall.UTF16PtrFromString(target(service, account))
	if err != nil {
		return err
	}
	ret, _, err := procCredDelete.Call(uintptr(unsafe.Pointer(targetName)), uintptr(credTypeGeneric), 0)
	if ret == 0 {
		return err
	}
	return nil
}
