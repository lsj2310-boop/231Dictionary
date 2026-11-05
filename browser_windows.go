//go:build windows

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	pWindowFromPoint           = user32.NewProc("WindowFromPoint")
	pGetWindowThreadProcessID  = user32.NewProc("GetWindowThreadProcessId")
	pGetWindowTextW            = user32.NewProc("GetWindowTextW")
	pGetAncestor               = user32.NewProc("GetAncestor")
	pOpenProcessForBrowser     = kernel32.NewProc("OpenProcess")
	pQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	browserExclusion           = true
	appSuppressed              = false
	browserProcessCache        = map[uint32]struct {
		name  string
		until time.Time
	}{}
	browserPDFMu    sync.Mutex
	browserPDFCache = make(map[uintptr]browserPDFResult)
)

type browserPDFResult struct {
	title   string
	isPDF   bool
	pending bool
	expires time.Time
}

const CMD_BROWSER_EXCLUDE = 221

func browserExclusionPath() string { return filepath.Join(dirPath(), "browser_exclusion_v31.txt") }
func loadBrowserExclusion() {
	b, err := os.ReadFile(browserExclusionPath())
	if err != nil {
		browserExclusion = true
		return
	}
	browserExclusion = browserExclusionFromFile(b)
}
func setBrowserExclusion(enabled bool) error {
	if err := os.MkdirAll(dirPath(), 0700); err != nil {
		return err
	}
	value := "off\n"
	if enabled {
		value = "on\n"
	}
	if err := os.WriteFile(browserExclusionPath(), []byte(value), 0600); err != nil {
		return err
	}
	browserExclusion = enabled
	return nil
}

func rootWindow(hwnd uintptr) uintptr {
	if hwnd == 0 {
		return 0
	}
	root, _, _ := pGetAncestor.Call(hwnd, 2) // GA_ROOT
	if root != 0 {
		return root
	}
	return hwnd
}
func windowCaption(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	b := make([]uint16, 1024)
	n, _, _ := pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(b[:n])
}

func browserPDFProbeScriptPath() string { return filepath.Join(dirPath(), "browser_pdf_probe.ps1") }

// UIA probe is asynchronous so the 60ms mouse timer is never blocked.
// The probe returns only a boolean; URL and document text aren't logged.
func startPDFProbe(hwnd uintptr, title string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3500*time.Millisecond)
		defer cancel()
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-ExecutionPolicy", "Bypass", "-File", browserPDFProbeScriptPath(), "-Hwnd", uintptrDecimal(hwnd))
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.Output()
		isPDF := err == nil && strings.EqualFold(strings.TrimSpace(string(out)), "PDF")
		browserPDFMu.Lock()
		// The tab may have changed during the probe. Don't use a stale result.
		if cached, ok := browserPDFCache[hwnd]; ok && cached.title == title {
			browserPDFCache[hwnd] = browserPDFResult{title: title, isPDF: isPDF, expires: time.Now().Add(25 * time.Second)}
		}
		browserPDFMu.Unlock()
	}()
}

func uintptrDecimal(hwnd uintptr) string {
	return strconv.FormatUint(uint64(hwnd), 10)
}

func browserWindowIsPDF(hwnd uintptr) bool {
	hwnd = rootWindow(hwnd)
	title := windowCaption(hwnd)
	if isPDFBrowserTitle(title) {
		return true
	}
	browserPDFMu.Lock()
	cached, ok := browserPDFCache[hwnd]
	if ok && cached.title == title && (cached.pending || time.Now().Before(cached.expires)) {
		result := cached.isPDF
		browserPDFMu.Unlock()
		return result
	}
	browserPDFCache[hwnd] = browserPDFResult{title: title, pending: true}
	// Prevent unbounded growth if many browser windows are opened/closed.
	if len(browserPDFCache) > 100 {
		browserPDFCache = map[uintptr]browserPDFResult{hwnd: {title: title, pending: true}}
	}
	browserPDFMu.Unlock()
	startPDFProbe(hwnd, title)
	return false
}

// Check both the actual window below the cursor and the foreground window.
// KakaoTalk is always excluded; web-page suppression remains user-selectable.
// All asynchronous translation completion paths must repeat this check.
func foregroundOrPointIsSuppressed(p point) bool {
	coords := uintptr(uint64(uint32(p.X)) | (uint64(uint32(p.Y)) << 32))
	child, _, _ := pWindowFromPoint.Call(coords)
	foreground, _, _ := pGetForegroundWindow.Call()
	candidates := []uintptr{rootWindow(foreground), rootWindow(child)}
	for _, hwnd := range candidates {
		name := executableForWindow(hwnd)
		if isKakaoTalkExecutable(name) {
			return true
		}
		if browserExclusion && isBrowserExecutable(name) && !browserWindowIsPDF(hwnd) {
			return true
		}
	}
	return false
}

// Compatibility wrapper for the browser behavior's existing tests/callers.
func foregroundOrPointIsBrowser(p point) bool { return foregroundOrPointIsSuppressed(p) }

func isBrowserProcessWindow(hwnd uintptr) bool { return isBrowserExecutable(executableForWindow(hwnd)) }

// Retrieves a process image name with a short PID cache. Do not use a window
// title: KakaoTalk chat titles and browser tab titles are user-controlled.
func executableForWindow(hwnd uintptr) string {
	if hwnd == 0 || hwnd == mainHWND || hwnd == tipHWND {
		return ""
	}
	var pid uint32
	pGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 || pid == uint32(os.Getpid()) {
		return ""
	}
	if cached, ok := browserProcessCache[pid]; ok && time.Now().Before(cached.until) {
		return cached.name
	}
	handle, _, _ := pOpenProcessForBrowser.Call(0x1000, 0, uintptr(pid))
	if handle == 0 {
		return ""
	}
	defer pCloseHandle.Call(handle)
	buf := make([]uint16, 32768)
	sz := uint32(len(buf))
	ok, _, _ := pQueryFullProcessImageName.Call(handle, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&sz)))
	if ok == 0 || sz == 0 || int(sz) > len(buf) {
		return ""
	}
	name := strings.ToLower(filepath.Base(syscall.UTF16ToString(buf[:sz])))
	browserProcessCache[pid] = struct {
		name  string
		until time.Time
	}{name, time.Now().Add(3 * time.Second)}
	return name
}
