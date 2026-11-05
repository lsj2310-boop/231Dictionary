//go:build !windows

package main

import (
	"context"
	"errors"
)

func windowsProxyWebRequest(_ context.Context, _, _ string, _ map[string]string, _ []byte) (webResponse, error) {
	return webResponse{}, errors.New("Windows proxy fallback only")
}
