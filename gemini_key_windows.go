//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	Size uint32
	Data *byte
}

var (
	cryptDLL      = syscall.NewLazyDLL("crypt32.dll")
	protectData   = cryptDLL.NewProc("CryptProtectData")
	unprotectData = cryptDLL.NewProc("CryptUnprotectData")
	localFree     = kernel32.NewProc("LocalFree")
)

func geminiKeyPath() string { return filepath.Join(dirPath(), "gemini_key.dpapi") }
func toBlob(p []byte) dataBlob {
	if len(p) == 0 {
		return dataBlob{}
	}
	return dataBlob{Size: uint32(len(p)), Data: &p[0]}
}
func encryptSecret(input string) ([]byte, error) {
	raw := []byte(input)
	if len(raw) == 0 {
		return nil, errors.New("Empty API key")
	}
	in := toBlob(raw)
	var out dataBlob
	r, _, e := protectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, fmt.Errorf("DPAPI encryption failed: %v", e)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.Data)))
	if out.Data == nil || out.Size == 0 {
		return nil, errors.New("DPAPI returned empty output")
	}
	return append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...), nil
}
func decryptSecret(encrypted []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}
	in := toBlob(encrypted)
	var out dataBlob
	r, _, e := unprotectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return "", fmt.Errorf("Could not decrypt Windows account key: %v", e)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.Data)))
	if out.Data == nil {
		return "", errors.New("Decryption returned empty output")
	}
	return string(unsafe.Slice(out.Data, int(out.Size))), nil
}
func saveGeminiKey(key string) error {
	key = strings.TrimSpace(key)
	if len(key) < 25 || len(key) > 512 {
		return errors.New("Invalid API key length. Enter a key issued by Google AI Studio")
	}
	secret, err := encryptSecret(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dirPath(), 0700); err != nil {
		return err
	}
	return os.WriteFile(geminiKeyPath(), secret, 0600)
}
func loadGeminiKey() (string, error) {
	raw, err := os.ReadFile(geminiKeyPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return decryptSecret(raw)
}
func geminiKeyRegistered() bool {
	k, e := loadGeminiKey()
	return e == nil && strings.TrimSpace(k) != ""
}
