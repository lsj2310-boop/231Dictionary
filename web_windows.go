//go:build windows

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

func webProxyScriptPath() string { return filepath.Join(dirPath(), "web_proxy.ps1") }
func windowsProxyWebRequest(ctx context.Context, method, endpoint string, headers map[string]string, body []byte) (webResponse, error) {
	payload, _ := json.Marshal(struct {
		Method, URL, Body string
		Headers           map[string]string `json:"headers"`
	}{Method: method, URL: endpoint, Body: base64.StdEncoding.EncodeToString(body), Headers: headers})
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", webProxyScriptPath())
	cmd.Stdin = bytes.NewReader(payload)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	raw, err := cmd.Output()
	if err != nil {
		return webResponse{}, fmt.Errorf("Windows proxy connection failed: %w", err)
	}
	var response struct {
		Status int    `json:"status"`
		Base64 string `json:"base64"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &response); err != nil {
		return webResponse{}, fmt.Errorf("Proxy response JSON error: %w", err)
	}
	if response.Status == 0 {
		return webResponse{}, errors.New(response.Error)
	}
	data, err := base64.StdEncoding.DecodeString(response.Base64)
	if err != nil {
		return webResponse{}, err
	}
	return webResponse{Status: response.Status, Body: data}, nil
}
func startProviderNetworkTest(provider string) {
	if testRunning {
		return
	}
	testRunning = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 13_000_000_000)
		defer cancel()
		var result string
		var err error
		result, err = googleTranslate(ctx, "Cab tilt system malfunction.")
		if err != nil {
			testText = provider + " connection failed: " + err.Error() + "\nCheck your company network or translation service policy."
		} else {
			testText = provider + " connection successful\nCab tilt system malfunction.\n→ " + result
		}
		appendDiagnostic(provider + " test: " + testText)
		pPostMessage.Call(mainHWND, WM_TEST_RESULT, 0, 0)
	}()
}
func openLastSentenceWeb() {
	if lastSelectedSentence == "" {
		callMessage("No English sentence has been selected recently.", MB_ICONWARNING)
		return
	}
	// Selected text goes to Google Translate via the default browser's URL.
	u := "https://translate.google.com/?sl=en&tl=ko&text=" + url.QueryEscape(lastSelectedSentence) + "&op=translate"
	pShellExecute.Call(0, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(u))), 0, 0, 1)
}
func showSourceInfo() {
	callMessage("Original translation implementation:\nhttps://github.com/ttop32/MouseTooltipTranslator\n\nAdapted request/response logic: src/translator/google.js and bing.js request/response logic\nMIT License © 2020 sang won kim (see LICENSE_MIT_MouseTooltipTranslator.txt)\n\nThe original is a Chrome extension; this app is an independent Windows implementation.\nUnofficial Google web translation endpoints may change or be blocked.\nAn API key is not required for the legacy web translation provider.", MB_ICONINFORMATION)
}
