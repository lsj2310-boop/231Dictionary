//go:build windows

// 231Dictionary for Windows 10/11 (amd64).
// Self-installing single binary, no admin access. Captures a small screen crop
// locally only when the pointer is still, for Windows built-in OCR fallback.
// Free for personal and corporate use. Created by Lee Sang Il.
package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

//go:embed ocr_worker.ps1 test_ocr.png web_proxy.ps1 browser_pdf_probe.ps1 LICENSE_MIT_MouseTooltipTranslator.txt
var embedded embed.FS

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	pRegisterClassEx        = user32.NewProc("RegisterClassExW")
	pCreateWindow           = user32.NewProc("CreateWindowExW")
	pDefWindow              = user32.NewProc("DefWindowProcW")
	pGetMessage             = user32.NewProc("GetMessageW")
	pTranslateMessage       = user32.NewProc("TranslateMessage")
	pDispatchMessage        = user32.NewProc("DispatchMessageW")
	pPostQuit               = user32.NewProc("PostQuitMessage")
	pPostMessage            = user32.NewProc("PostMessageW")
	pSetTimer               = user32.NewProc("SetTimer")
	pGetCursor              = user32.NewProc("GetCursorPos")
	pGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pShowWindow             = user32.NewProc("ShowWindow")
	pInvalidate             = user32.NewProc("InvalidateRect")
	pBeginPaint             = user32.NewProc("BeginPaint")
	pEndPaint               = user32.NewProc("EndPaint")
	pFillRect               = user32.NewProc("FillRect")
	pFrameRect              = user32.NewProc("FrameRect")
	pDrawText               = user32.NewProc("DrawTextW")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pCreateFont             = gdi32.NewProc("CreateFontW")
	pCreateBrush            = gdi32.NewProc("CreateSolidBrush")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pGetDC                  = user32.NewProc("GetDC")
	pReleaseDC              = user32.NewProc("ReleaseDC")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pGetDIBits              = gdi32.NewProc("GetDIBits")
	pStretchBlt             = gdi32.NewProc("StretchBlt")
	pSetStretchBltMode      = gdi32.NewProc("SetStretchBltMode")
	pGetTextExtent          = gdi32.NewProc("GetTextExtentPoint32W")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")
	pGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	pLoadIcon               = user32.NewProc("LoadIconW")
	pShellNotify            = shell32.NewProc("Shell_NotifyIconW")
	pCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	pAppendMenu             = user32.NewProc("AppendMenuW")
	pTrackPopup             = user32.NewProc("TrackPopupMenu")
	pDestroyMenu            = user32.NewProc("DestroyMenu")
	pSetForeground          = user32.NewProc("SetForegroundWindow")
	pMessageBox             = user32.NewProc("MessageBoxW")
	pCreateMutex            = kernel32.NewProc("CreateMutexW")
	pCloseHandle            = kernel32.NewProc("CloseHandle")
	pGetLastError           = kernel32.NewProc("GetLastError")
	pGetModuleHandle        = kernel32.NewProc("GetModuleHandleW")
	pDPIAware               = user32.NewProc("SetProcessDPIAware")
	pFindWindow             = user32.NewProc("FindWindowW")
	pRegisterHotKey         = user32.NewProc("RegisterHotKey")
	pUnregisterHotKey       = user32.NewProc("UnregisterHotKey")
	pShellExecute           = shell32.NewProc("ShellExecuteW")
)

const (
	WM_DESTROY           = 0x0002
	WM_PAINT             = 0x000F
	WM_CLOSE             = 0x0010
	WM_COMMAND           = 0x0111
	WM_TIMER             = 0x0113
	WM_CONTEXTMENU       = 0x007B
	WM_APP               = 0x8000
	WM_TRAY              = WM_APP + 10
	WM_RESULT            = WM_APP + 11
	WM_TEST_RESULT       = WM_APP + 12
	WM_TRANSLATED        = WM_APP + 13
	DT_WORDBREAK         = 0x10
	DT_CALCRECT          = 0x400
	WM_HOTKEY            = 0x0312
	MOD_CONTROL          = 0x0002
	MOD_ALT              = 0x0001
	MOD_SHIFT            = 0x0004
	WM_LBUTTONDBLCLK     = 0x0203
	WM_RBUTTONUP         = 0x0205
	WS_POPUP             = 0x80000000
	WS_EX_TOPMOST        = 0x00000008
	WS_EX_TRANSPARENT    = 0x00000020
	WS_EX_TOOLWINDOW     = 0x00000080
	WS_EX_NOACTIVATE     = 0x08000000
	SW_HIDE              = 0
	SW_SHOWNOACTIVATE    = 4
	SWP_NOACTIVATE       = 0x0010
	SWP_NOMOVE           = 0x0002
	SWP_SHOWWINDOW       = 0x0040
	HWND_TOPMOST         = ^uintptr(0)
	NIM_ADD              = 0x0
	NIM_DELETE           = 0x2
	NIF_MESSAGE          = 0x1
	NIF_ICON             = 0x2
	NIF_TIP              = 0x4
	TPM_RIGHTBUTTON      = 0x2
	TPM_RETURNCMD        = 0x100
	MF_STRING            = 0x0
	MF_CHECKED           = 0x8
	CMD_TOGGLE           = 201
	CMD_STARTUP          = 202
	CMD_INFO             = 203
	CMD_UNINSTALL        = 204
	CMD_EXIT             = 205
	CMD_ABOUT            = 206
	CMD_SELFTEST         = 207
	CMD_LOG              = 208
	CMD_DICTIONARY_TEST  = 209
	CMD_NETWORK_TEST     = 210
	CMD_CUSTOM_EDIT      = 211
	CMD_CUSTOM_RELOAD    = 212
	CMD_TOOLTIP_TEST     = 213
	CMD_GOOGLE_TEST      = 216
	CMD_BING_TEST        = 217
	CMD_OPEN_WEB         = 218
	CMD_SOURCE_INFO      = 219
	CMD_GEMINI_KEY       = 220
	WM_GEMINI_KEY_PROMPT = WM_APP + 17
	MB_YESNO             = 0x4
	MB_ICONQUESTION      = 0x20
	MB_ICONINFORMATION   = 0x40
	MB_ICONWARNING       = 0x30
	IDYES                = 6
	DT_LEFT              = 0
	DT_SINGLELINE        = 0x20
	DT_NOPREFIX          = 0x800
	DT_END_ELLIPSIS      = 0x8000
	LOGPIXELSY           = 90
	HALFTONE             = 4
	HOVER_DELAY          = 200 * time.Millisecond
	TOOLTIP_MAX_WIDTH    = 260
	TOOLTIP_MIN_WIDTH    = 78
	TRANSPARENT          = 1
	SM_XVIRTUALSCREEN    = 76
	SM_YVIRTUALSCREEN    = 77
	SM_CXVIRTUALSCREEN   = 78
	SM_CYVIRTUALSCREEN   = 79
	SRCCOPY              = 0x00CC0020
	BI_RGB               = 0
	DIB_RGB_COLORS       = 0
)

type point struct{ X, Y int32 }
type size struct{ CX, CY int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct {
	Hwnd           uintptr
	Message        uint32
	_              uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	LPrivate       uint32
}
type wndClassEx struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSmall                          uintptr
}
type paintStruct struct {
	Hdc                uintptr
	Erase              int32
	Paint              rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type bitmapInfoHeader struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}
type rgbQuad struct{ B, G, R, A byte }
type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]rgbQuad
}
type notifyIconData struct {
	CbSize           uint32
	Hwnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State, StateMask uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUID             [16]byte
	BalloonIcon      uintptr
}

type workerAnswer struct {
	Sentence  string  `json:"sentence"`
	OK        bool    `json:"ok"`
	Word      string  `json:"word"`
	Backend   string  `json:"backend"`
	OCR       bool    `json:"ocr"`
	OCRError  string  `json:"ocrError"`
	Available string  `json:"available"`
	Count     int     `json:"count"`
	Distance  float64 `json:"distance"`
	Error     string  `json:"error"`
}
type hoverResult struct {
	P                      point
	Word, Meaning, Backend string
	TranslationSource      string
	OCR                    bool
	OCRError               string
	Err                    string
	Available              string
	Count                  int
	Distance               float64
}

var (
	mainHWND, tipHWND                           uintptr
	uiMu                                        sync.Mutex
	result                                      hoverResult
	tipWord, tipMean                            string
	tipWidth, tipHeight                         = 160, 38
	lastDiagnostic                              string
	tipShown                                    bool
	active                                      = true
	working                                     = false
	testRunning                                 = false
	testText                                    string
	lastSuccess                                 = false
	manualDebug                                 = false
	lastPos                                     point
	lastMoved                                   time.Time
	lastChecked                                 point
	lastCheckAt                                 time.Time
	fontWord, fontMeaning, brushBG, brushBorder uintptr
	fontPixelHeight                             = 11
	mutexHandle                                 uintptr
	ocrEngine                                   *ocrWorker
	hoverGeneration                             uint64
	completedTranslation                        translatedHover
	validEnglish                                = regexp.MustCompile(`^[A-Za-z][A-Za-z'-]{1,38}$`)
)

type translatedHover struct {
	RequestID uint64
	P         point
	Word      string
	Meaning   string
	Source    string
	Err       string
}

func ptr(s string) *uint16         { return syscall.StringToUTF16Ptr(s) }
func color(r, g, b uint32) uintptr { return uintptr(r | g<<8 | b<<16) }
func callMessage(s string, flags uintptr) uintptr {
	r, _, _ := pMessageBox.Call(0, uintptr(unsafe.Pointer(ptr(s))), uintptr(unsafe.Pointer(ptr("231Dictionary"))), flags)
	return r
}
func dirPath() string     { return filepath.Join(os.Getenv("LOCALAPPDATA"), "231Dictionary") }
func exePath() string     { return filepath.Join(dirPath(), "231Dictionary.exe") }
func ocrPath() string     { return filepath.Join(dirPath(), "ocr_worker.ps1") }
func startupName() string { return "231Dictionary" }
func logPath() string     { return filepath.Join(dirPath(), "diagnostic.log") }
func readyPath() string   { return filepath.Join(dirPath(), "v10_ready.status") }
func appendDiagnostic(msg string) {
	if err := os.MkdirAll(dirPath(), 0755); err != nil {
		return
	}
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	if info, e := f.Stat(); e == nil && info.Size() > 1024*1024 {
		_ = os.WriteFile(logPath(), []byte("Log reset due to size limit\n"), 0644)
	}
}
func installed() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	a, _ := filepath.Abs(exe)
	b, _ := filepath.Abs(exePath())
	return strings.EqualFold(a, b)
}
func ensureWorker() error {
	data, err := embedded.ReadFile("ocr_worker.ps1")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dirPath(), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(ocrPath(), data, 0644); err != nil {
		return err
	}
	script, e := embedded.ReadFile("web_proxy.ps1")
	if e != nil {
		return e
	}
	if e := os.WriteFile(webProxyScriptPath(), script, 0600); e != nil {
		return e
	}
	pdfProbe, e := embedded.ReadFile("browser_pdf_probe.ps1")
	if e != nil {
		return e
	}
	if e := os.WriteFile(browserPDFProbeScriptPath(), pdfProbe, 0600); e != nil {
		return e
	}
	license, e := embedded.ReadFile("LICENSE_MIT_MouseTooltipTranslator.txt")
	if e != nil {
		return e
	}
	if e := os.WriteFile(filepath.Join(dirPath(), "LICENSE_MIT_MouseTooltipTranslator.txt"), license, 0644); e != nil {
		return e
	}
	return nil
}

func startupEnabled() bool {
	cmd := exec.Command("reg.exe", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", startupName())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}
func configureStartup(enable bool) {
	var cmd *exec.Cmd
	if enable {
		cmd = exec.Command("reg.exe", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", startupName(), "/t", "REG_SZ", "/d", `"`+exePath()+`" -run`, "/f")
	} else {
		cmd = exec.Command("reg.exe", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", startupName(), "/f")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}
func testImagePath() string { return filepath.Join(dirPath(), "test_ocr.png") }
func extractTestImage() error {
	b, e := embedded.ReadFile("test_ocr.png")
	if e != nil {
		return e
	}
	return os.WriteFile(testImagePath(), b, 0644)
}
func shortcutPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "231Dictionary.lnk")
}
func addShortcut() error {
	// Windows Script Host creates a native Start Menu .lnk file.
	cmdtext := `$s=(New-Object -ComObject WScript.Shell).CreateShortcut('` + strings.ReplaceAll(shortcutPath(), "'", "''") + `');` +
		`$s.TargetPath='` + strings.ReplaceAll(exePath(), "'", "''") + `';$s.Arguments='-run';$s.WorkingDirectory='` + strings.ReplaceAll(dirPath(), "'", "''") + `';` +
		`$s.Description='Created by Lee Sang Il | Free for personal and business use';$s.Save()`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmdtext)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
func installSelf() {
	if callMessage("Install or update 231Dictionary v1.0?\n\nHover over an English word to view its meaning.\nDrag-select English or Korean sentences for Gemini translation, with source line breaks preserved.\nOrdinary browser web pages are excluded, but PDFs in browsers are supported.\nKakaoTalk is always excluded.\nYour saved Gemini API key and preferences will be retained.\n\nCreated by Lee Sang Il. Free for personal and business use.", MB_YESNO|MB_ICONQUESTION) != IDYES {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		callMessage(err.Error(), MB_ICONWARNING)
		return
	}
	if err = os.MkdirAll(dirPath(), 0755); err != nil {
		callMessage(err.Error(), MB_ICONWARNING)
		return
	}
	// A running Windows EXE cannot be overwritten. Stage the update, tell the
	// previous tray process to close, and atomically swap once it exits.
	staged := exePath() + ".updating"
	in, err := os.Open(exe)
	if err != nil {
		callMessage(err.Error(), MB_ICONWARNING)
		return
	}
	defer in.Close()
	out, err := os.Create(staged)
	if err != nil {
		callMessage("Could not create update file: "+err.Error(), MB_ICONWARNING)
		return
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(staged)
		callMessage("Could not copy update file: "+err.Error(), MB_ICONWARNING)
		return
	}
	existing, _, _ := pFindWindow.Call(uintptr(unsafe.Pointer(ptr("231DictionaryWindow"))), 0)
	if existing != 0 {
		pPostMessage.Call(existing, WM_CLOSE, 0, 0)
	}
	// Windows cannot rename directly over an existing executable. Move the old
	// closed binary to a backup first, then move the staged one into place.
	// On errors, keep/restore the previous working installation.
	backup := exePath() + ".previous"
	_ = os.Remove(backup)
	var swapErr error
	for attempt := 0; attempt < 85; attempt++ {
		if _, e := os.Stat(exePath()); errors.Is(e, os.ErrNotExist) {
			swapErr = nil
			break
		}
		swapErr = os.Rename(exePath(), backup)
		if swapErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if swapErr != nil {
		_ = os.Remove(staged)
		callMessage("Update failed: the previous version may be running or its file is locked.\nExit the existing tray app and try again.\n"+swapErr.Error(), MB_ICONWARNING)
		return
	}
	if err = os.Rename(staged, exePath()); err != nil {
		_ = os.Rename(backup, exePath()) // rollback
		callMessage("Could not replace the program file. The previous version was restored.\n"+err.Error(), MB_ICONWARNING)
		return
	}
	prepareDictionary()
	requestConsent() // Ask before sending selected text to public web services.
	if err = ensureWorker(); err != nil {
		callMessage(err.Error(), MB_ICONWARNING)
		return
	}
	_ = addShortcut()
	if callMessage("Installation is complete.\n\nStart 231Dictionary automatically when you sign in to Windows?\n(You can change this from the tray menu.)", MB_YESNO|MB_ICONQUESTION) == IDYES {
		configureStartup(true)
	} else {
		configureStartup(false)
	}
	// Don't show a false success if Windows security or a previous instance
	// prevents the newly installed version from opening its tray window.
	_ = os.Remove(readyPath())
	launch := exec.Command(exePath(), "-run")
	launch.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err = launch.Start(); err != nil {
		callMessage("Installed files, but failed to launch v1.0: "+err.Error(), MB_ICONWARNING)
		return
	}
	ready := false
	for tries := 0; tries < 50; tries++ {
		b, e := os.ReadFile(readyPath())
		if e == nil && strings.HasPrefix(string(b), "v1.0 ") {
			ready = true
			break
		}
		time.Sleep(120 * time.Millisecond)
	}
	if !ready {
		callMessage("The update files were copied, but the newly installed v1.0 app could not be confirmed running.\n\nExit the existing app and reinstall, or check your security software.\n\nDiagnostic log: "+logPath(), MB_ICONWARNING)
		return
	}
	keyNote := "If you have already saved your Gemini API key, you do not need to enter it again."
	if !geminiKeyRegistered() {
		keyNote = "Gemini API key not configured. Use Set / Change Gemini API Key in the system tray."
	}
	callMessage("231Dictionary v1.0 installed successfully!\n\nSelected sentences are translated in both directions (English/Korean), preserving source line breaks.\nKey terms and expressions appear below the translation.\nBrowser web pages are excluded by default, browser PDFs are permitted, and KakaoTalk is always excluded.\n"+keyNote+"\n\nCreated by Lee Sang Il. Free for personal and business use.", MB_ICONINFORMATION)
}
func uninstallSelf() {
	if callMessage("Uninstall 231Dictionary?\n\nThis also removes the startup entry and the Start Menu shortcut.", MB_YESNO|MB_ICONQUESTION) != IDYES {
		return
	}
	configureStartup(false)
	_ = os.Remove(shortcutPath())
	_ = os.Remove(ocrPath())
	// The currently running exe is locked until the app exits. Delay deletion.
	cleanup := `ping 127.0.0.1 -n 3 >nul & del /f /q "` + exePath() + `" & rmdir /s /q "` + dirPath() + `"`
	c := exec.Command("cmd.exe", "/C", cleanup)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = c.Start()
	cleanupTray()
	pPostQuit.Call(0)
}

func makeWindow() error {
	pDPIAware.Call()
	cls := ptr("231DictionaryWindow")
	instance, _, _ := pGetModuleHandle.Call(0)
	icon, _, _ := pLoadIcon.Call(0, uintptr(32516)) // IDI_INFORMATION
	bg, _, _ := pCreateBrush.Call(color(255, 252, 242))
	bd, _, _ := pCreateBrush.Call(color(205, 212, 218))
	brushBG, brushBorder = bg, bd
	// Convert user-requested 8pt typography to actual Windows font pixels.
	hdc, _, _ := pGetDC.Call(0)
	dpi := uintptr(96)
	if hdc != 0 {
		if d, _, _ := pGetDeviceCaps.Call(hdc, LOGPIXELSY); d >= 72 && d <= 600 {
			dpi = d
		}
		pReleaseDC.Call(0, hdc)
	}
	fontPixelHeight = int((dpi*8 + 36) / 72)
	fontWordHeight := int32(-fontPixelHeight)
	fontMeaningHeight := int32(-fontPixelHeight)
	fontWord, _, _ = pCreateFont.Call(uintptr(fontWordHeight), 0, 0, 0, 700, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
	fontMeaning, _, _ = pCreateFont.Call(uintptr(fontMeaningHeight), 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(ptr("Malgun Gothic"))))
	cb := syscall.NewCallback(windowProc)
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), Proc: cb, Instance: instance, Icon: icon, ClassName: cls}
	r, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return fmt.Errorf("RegisterClassEx: %v", err)
	}
	mainHWND, _, err = pCreateWindow.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(ptr("231Dictionary"))), 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if mainHWND == 0 {
		return fmt.Errorf("Create main window: %v", err)
	}
	tipHWND, _, err = pCreateWindow.Call(WS_EX_TOPMOST|WS_EX_TRANSPARENT|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(ptr("231Dictionary"))), WS_POPUP, 0, 0, uintptr(tipWidth), uintptr(tipHeight), mainHWND, 0, instance, 0)
	if tipHWND == 0 {
		return fmt.Errorf("Create tooltip: %v", err)
	}
	pShowWindow.Call(tipHWND, SW_HIDE)
	addTray(icon)
	pSetTimer.Call(mainHWND, 1, 60, 0)
	if ok, _, _ := pRegisterHotKey.Call(mainHWND, 1, MOD_CONTROL|MOD_ALT|MOD_SHIFT, uintptr('D')); ok == 0 {
		appendDiagnostic("register hotkey failure; use tray menu diagnostics")
	}
	return nil
}
func addTray(icon uintptr) {
	var ni notifyIconData
	ni.CbSize = uint32(unsafe.Sizeof(ni))
	ni.Hwnd = mainHWND
	ni.ID = 1
	ni.Flags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	ni.CallbackMessage = WM_TRAY
	ni.Icon = icon
	copy(ni.Tip[:], syscall.StringToUTF16("231Dictionary | Lee Sang Il | Free personal and business use"))
	pShellNotify.Call(NIM_ADD, uintptr(unsafe.Pointer(&ni)))
}
func cleanupTray() {
	var ni notifyIconData
	ni.CbSize = uint32(unsafe.Sizeof(ni))
	ni.Hwnd = mainHWND
	ni.ID = 1
	pShellNotify.Call(NIM_DELETE, uintptr(unsafe.Pointer(&ni)))
}
func menu() {
	m, _, _ := pCreatePopupMenu.Call()
	flag := uintptr(MF_STRING)
	if active {
		flag |= MF_CHECKED
	}
	pAppendMenu.Call(m, flag, CMD_TOGGLE, uintptr(unsafe.Pointer(ptr("Enable / Disable Word Tooltips"))))
	flag = MF_STRING
	if startupEnabled() {
		flag |= MF_CHECKED
	}
	pAppendMenu.Call(m, flag, CMD_STARTUP, uintptr(unsafe.Pointer(ptr("Start with Windows"))))
	pAppendMenu.Call(m, MF_STRING, CMD_INFO, uintptr(unsafe.Pointer(ptr("Help and Status"))))
	pAppendMenu.Call(m, MF_STRING, CMD_ABOUT, uintptr(unsafe.Pointer(ptr("About / Author / License"))))
	pAppendMenu.Call(m, MF_STRING, CMD_GEMINI_KEY, uintptr(unsafe.Pointer(ptr("Set / Change Gemini API Key"))))
	// Browser exclusion affects both 0.2s word tooltip and drag selection.
	browserFlag := uintptr(MF_STRING)
	if browserExclusion {
		browserFlag |= MF_CHECKED
	}
	pAppendMenu.Call(m, browserFlag, CMD_BROWSER_EXCLUDE, uintptr(unsafe.Pointer(ptr("Disable on Web Pages (Allow PDFs)"))))
	consentFlag := uintptr(MF_STRING)
	if sentenceOnlineAllowed {
		consentFlag |= MF_CHECKED
	}
	pAppendMenu.Call(m, consentFlag, CMD_SENTENCE_ONLINE, uintptr(unsafe.Pointer(ptr("Allow Online Translation"))))
	pAppendMenu.Call(m, MF_STRING, CMD_UNINSTALL, uintptr(unsafe.Pointer(ptr("Uninstall 231Dictionary"))))
	pAppendMenu.Call(m, MF_STRING, CMD_EXIT, uintptr(unsafe.Pointer(ptr("Exit"))))
	var p point
	pGetCursor.Call(uintptr(unsafe.Pointer(&p)))
	pSetForeground.Call(mainHWND)
	cmd, _, _ := pTrackPopup.Call(m, TPM_RIGHTBUTTON|TPM_RETURNCMD, uintptr(p.X), uintptr(p.Y), 0, mainHWND, 0)
	pDestroyMenu.Call(m)
	switch cmd {
	case CMD_TOGGLE:
		active = !active
		sentenceShown = false
		hide231Dictionary()
	case CMD_STARTUP:
		configureStartup(!startupEnabled())
	case CMD_INFO:
		showInfo()
	case CMD_ABOUT:
		showAbout()
	case CMD_GEMINI_KEY:
		openGeminiKeyDialog()
	case CMD_BROWSER_EXCLUDE:
		if err := setBrowserExclusion(!browserExclusion); err != nil {
			callMessage("Failed to save browser setting: "+err.Error(), MB_ICONWARNING)
		} else {
			// Invalidate in-flight screen/translation results when policy changes.
			hoverGeneration++
			selectionSequence++
			dragDown, dragQualified, selectionBusy, sentenceShown = false, false, false, false
			hide231Dictionary()
			lastMoved = time.Now()
			lastCheckAt = time.Time{}
			appSuppressed = false
		}
	case CMD_SELFTEST:
		startOCRSelftest()
	case CMD_LOG:
		openDiagnosticLog()
	case CMD_DICTIONARY_TEST:
		testOfflineDictionary()
	case CMD_TOOLTIP_TEST:
		test231DictionaryDisplay()
	case CMD_SENTENCE_TEST:
		showSentenceTip("The software update requires a root cause analysis.", "소프트웨어 업데이트에는 근본 원인 분석이 필요합니다.", extractKeyTerms("The software update requires a root cause analysis.", 6), mouse())
	case CMD_SENTENCE_ONLINE:
		if sentenceOnlineAllowed {
			setSentenceConsent(false)
		} else {
			requestConsent()
		}
	case CMD_GOOGLE_TEST:
		startProviderNetworkTest("Google GTX")
	case CMD_BING_TEST:
		startProviderNetworkTest("Bing")
	case CMD_OPEN_WEB:
		openLastSentenceWeb()
	case CMD_SOURCE_INFO:
		showSourceInfo()
	case CMD_NETWORK_TEST:
		startNetworkTest()
	case CMD_CUSTOM_EDIT:
		openCustomDictionary()
	case CMD_CUSTOM_RELOAD:
		n, err := reloadCustomDictionary()
		if err != nil {
			callMessage("Failed to load custom dictionary: "+err.Error(), MB_ICONWARNING)
		} else {
			callMessage(fmt.Sprintf("Loaded %d custom dictionary entries", n), MB_ICONINFORMATION)
		}
	case CMD_UNINSTALL:
		uninstallSelf()
	case CMD_EXIT:
		cleanupTray()
		pPostQuit.Call(0)
	}
}
func showAbout() {
	callMessage("231Dictionary v1.0\n\nCreated by Lee Sang Il (independent developer)\nFree for personal and business use\n\nFeatures:\n- Hover over English words for Gemini-powered word meanings.\n- Drag-select English/Korean sentences for two-way translation.\n- Preserve source line breaks and show key terms.\n- Exclude ordinary web pages (allow PDFs in browsers).\n- Always exclude KakaoTalk, including its chat windows.\n\nYour Gemini API key is encrypted for your Windows user account.\nSelected text may be sent to Google Gemini. Do not select confidential information.\n\nThis is not official TDM software.", MB_ICONINFORMATION)
}
func showInfo() {
	tooltipState := "On"
	if !active {
		tooltipState = "Off"
	}
	keyState := "Not configured (select Set / Change Gemini API Key)"
	if geminiKeyRegistered() {
		keyState = "Configured (encrypted per Windows user)"
	}
	onlineState := "Off"
	if sentenceOnlineAllowed {
		onlineState = "On"
	}
	browserState := "On (web pages excluded; browser PDFs allowed)"
	if !browserExclusion {
		browserState = "Off (browser pages enabled)"
	}
	callMessage("231Dictionary v1.0\n\nWord tooltips: "+tooltipState+"\nGemini API key: "+keyState+"\nOnline sentence translation: "+onlineState+"\nBrowser exclusion: "+browserState+"\nKakaoTalk exclusion: Always on\n\n1. Hover over an English word for its meaning.\n2. Drag-select English or Korean text for translation with key terms.\n3. In PowerPoint, select text inside a text box; in PDF, use the text-selection tool.\n4. Your API key is verified when it is saved.\n\nCreated by Lee Sang Il", MB_ICONINFORMATION)
}

func windowProc(hwnd, message, wp, lp uintptr) uintptr {
	if hwnd == tipHWND && message == WM_PAINT {
		drawTip(hwnd)
		return 0
	}
	if hwnd == mainHWND {
		switch message {
		case WM_TIMER:
			onTimer()
			return 0
		case WM_RESULT:
			handleResult()
			return 0
		case WM_TEST_RESULT:
			testRunning = false
			uiMu.Lock()
			testMessage := testText
			uiMu.Unlock()
			callMessage(testMessage, MB_ICONINFORMATION)
			return 0
		case WM_TRANSLATED:
			handleTranslated()
			return 0
		case WM_SELECTION:
			handleSelection()
			return 0
		case WM_SENTENCE_TRANSLATED:
			handleSentenceTranslation()
			return 0
		case WM_HOTKEY:
			if wp == 1 {
				manualDebug = true
				lastCheckAt = time.Time{}
				lastMoved = time.Now().Add(-HOVER_DELAY)
				hide231Dictionary()
				onTimer()
			}
			return 0
		case WM_GEMINI_KEY_PROMPT:
			if !geminiKeyRegistered() {
				openGeminiKeyDialog()
			}
			return 0
		case WM_TRAY:
			if lp == WM_RBUTTONUP {
				menu()
			} else if lp == WM_LBUTTONDBLCLK {
				showInfo()
			}
			return 0
		case WM_DESTROY:
			pUnregisterHotKey.Call(mainHWND, 1)
			cleanupTray()
			pPostQuit.Call(0)
			return 0
		}
	}
	r, _, _ := pDefWindow.Call(hwnd, uintptr(message), wp, lp)
	return r
}
func drawTip(hwnd uintptr) {
	if tipMode == "sentence" {
		drawSentenceTip(hwnd)
		return
	}
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	uiMu.Lock()
	w, m, width, height := tipWord, tipMean, tipWidth, tipHeight
	uiMu.Unlock()
	all := rect{0, 0, int32(width), int32(height)}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&all)), brushBG)
	pFrameRect.Call(hdc, uintptr(unsafe.Pointer(&all)), brushBorder)
	pSetBkMode.Call(hdc, TRANSPARENT)
	pSetTextColor.Call(hdc, color(34, 55, 75))
	pSelectObject.Call(hdc, fontWord)
	line := int32(fontPixelHeight + 4)
	wordRect := rect{7, 3, int32(width) - 7, 3 + line}
	drawUnicode(hdc, w, &wordRect)
	pSetTextColor.Call(hdc, color(35, 104, 80))
	pSelectObject.Call(hdc, fontMeaning)
	meaningRect := rect{7, 3 + line, int32(width) - 7, int32(height) - 3}
	drawUnicode(hdc, m, &meaningRect)
	pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}
func drawUnicode(hdc uintptr, s string, r *rect) {
	u := syscall.StringToUTF16(s)
	if len(u) < 2 {
		return
	}
	pDrawText.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(r)), DT_LEFT|DT_SINGLELINE|DT_NOPREFIX|DT_END_ELLIPSIS)
}
func measureText(hdc, font uintptr, s string) int {
	if s == "" {
		return 0
	}
	old, _, _ := pSelectObject.Call(hdc, font)
	defer pSelectObject.Call(hdc, old)
	u := syscall.StringToUTF16(s)
	var sz size
	ok, _, _ := pGetTextExtent.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&sz)))
	if ok == 0 {
		return len([]rune(s)) * fontPixelHeight
	}
	return int(sz.CX)
}
func tooltipSize(word, meaning string) (int, int) {
	hdc, _, _ := pGetDC.Call(0)
	width := TOOLTIP_MIN_WIDTH
	if hdc != 0 {
		width = max(width, max(measureText(hdc, fontWord, word), measureText(hdc, fontMeaning, meaning))+16)
		pReleaseDC.Call(0, hdc)
	}
	if width > TOOLTIP_MAX_WIDTH {
		width = TOOLTIP_MAX_WIDTH
	}
	height := 2*(fontPixelHeight+4) + 6
	return width, height
}
func hide231Dictionary() {
	if tipShown {
		pShowWindow.Call(tipHWND, SW_HIDE)
		tipShown = false
	}
}
func mouse() point { var p point; pGetCursor.Call(uintptr(unsafe.Pointer(&p))); return p }
func near(a, b point) bool {
	dx := int64(a.X) - int64(b.X)
	if dx < 0 {
		dx = -dx
	}
	dy := int64(a.Y) - int64(b.Y)
	if dy < 0 {
		dy = -dy
	}
	return dx <= 4 && dy <= 4
}
func onTimer() {
	if !active {
		hide231Dictionary()
		return
	}
	now := time.Now()
	p := mouse()
	if foregroundOrPointIsSuppressed(p) {
		if !appSuppressed {
			appSuppressed = true
			// A completed asynchronous translation must never reappear in a browser.
			hoverGeneration++
			selectionSequence++
			dragDown, dragQualified, selectionBusy, sentenceShown = false, false, false, false
			hide231Dictionary()
			lastCheckAt = time.Time{}
		}
		lastPos, lastMoved = p, now
		return
	}
	if appSuppressed {
		appSuppressed = false
		lastPos, lastMoved = p, now
		lastCheckAt = time.Time{}
		return // Restart the normal 0.2-second dwell after leaving a browser.
	}
	if observeDrag(p, now) {
		return
	}
	if !near(p, lastPos) {
		lastPos = p
		lastMoved = now
		hide231Dictionary()
		return
	}
	if lastMoved.IsZero() {
		lastMoved = now
		return
	}
	if now.Sub(lastMoved) < HOVER_DELAY {
		return
	}
	if working {
		return
	}
	if near(p, lastChecked) && !lastCheckAt.IsZero() {
		// v1.1 bug: a failed hover was suppressed FOREVER until the mouse moved.
		// Retry no-match every 1.8s; do not repeatedly replace visible good tooltips.
		if tipShown {
			return
		}
		if now.Sub(lastCheckAt) < 1800*time.Millisecond {
			return
		}
	}
	lastChecked = p
	lastCheckAt = now
	working = true
	hide231Dictionary()
	go func(pos point) {
		r := analyze(pos)
		uiMu.Lock()
		result = r
		uiMu.Unlock()
		pPostMessage.Call(mainHWND, WM_RESULT, 0, 0)
	}(p)
}
func handleResult() {
	working = false
	uiMu.Lock()
	r := result
	hoverGeneration++
	requestID := hoverGeneration
	if r.Err != "" {
		lastDiagnostic = "Recognition error: " + r.Err
	} else if r.OCRError != "" && !r.OCR && r.Word == "" {
		lastDiagnostic = "OCR unavailable: " + r.OCRError + " | " + r.Available
	} else if r.Word == "" {
		lastDiagnostic = fmt.Sprintf("No word recognized (%s, OCR candidates %d, distance %.1f px, %s)", r.Backend, r.Count, r.Distance, r.Available)
	} else if r.Meaning == "" {
		lastDiagnostic = "English recognition succeeded (" + r.Backend + ", " + r.Word + "); looking up meaning"
	} else {
		lastDiagnostic = "Recognition succeeded (" + r.Backend + ", " + r.Word + ", " + r.TranslationSource + ")"
	}
	diagnostic := lastDiagnostic
	uiMu.Unlock()
	appendDiagnostic(diagnostic)
	if manualDebug {
		manualDebug = false
		// Dialog is intentionally diagnostic; it may cover the hovered window.
		callMessage(diagnostic+"\n\nFor details, check diagnostic.log in the app folder.", MB_ICONINFORMATION)
	}
	if !active || sentencePriorityActive(dragDown, selectionBusy, sentenceShown) || foregroundOrPointIsSuppressed(mouse()) || !near(mouse(), r.P) || r.Word == "" {
		return
	}
	meaning := r.Meaning
	if meaning == "" {
		meaning = "Looking up meaning…"
	}
	showSmall231Dictionary(r.Word, meaning, r.P)
	if r.Meaning == "" {
		// Network calls are NEVER in the OCR loop: show the word immediately.
		go func(id uint64, pos point, word string) {
			translated := translateDetailed(word)
			uiMu.Lock()
			completedTranslation = translatedHover{
				RequestID: id, P: pos, Word: word,
				Meaning: translated.Meaning, Source: translated.Source, Err: translated.Err,
			}
			uiMu.Unlock()
			pPostMessage.Call(mainHWND, WM_TRANSLATED, 0, 0)
		}(requestID, r.P, r.Word)
	}
}

// Repaint only when the translated result still belongs to the current hover.
func handleTranslated() {
	uiMu.Lock()
	t := completedTranslation
	currentID := hoverGeneration
	uiMu.Unlock()
	if t.RequestID != currentID || sentencePriorityActive(dragDown, selectionBusy, sentenceShown) || !tipShown || !active || foregroundOrPointIsSuppressed(mouse()) || !near(mouse(), t.P) {
		return
	}
	meaning := t.Meaning
	if meaning == "" {
		meaning = "Meaning unavailable (check connection)"
	}
	showSmall231Dictionary(t.Word, meaning, t.P)
	if t.Err != "" {
		appendDiagnostic("Online lookup failed for " + t.Word + ": " + t.Err)
	} else {
		appendDiagnostic("Online translation success for " + t.Word + " using " + t.Source)
	}
}

// A single UI path is used by offline, online, and independent popup tests.
func showSmall231Dictionary(word, meaning string, at point) {
	if sentencePriorityActive(dragDown, selectionBusy, sentenceShown) || foregroundOrPointIsSuppressed(mouse()) {
		return
	}
	tipMode = "word"
	sentenceShown = false
	uiMu.Lock()
	tipWord, tipMean = word, meaning
	tipWidth, tipHeight = tooltipSize(word, meaning)
	w, h := tipWidth, tipHeight
	uiMu.Unlock()
	x, y := int(at.X)+14, int(at.Y)+17
	minX, _, _ := pGetSystemMetrics.Call(SM_XVIRTUALSCREEN)
	minY, _, _ := pGetSystemMetrics.Call(SM_YVIRTUALSCREEN)
	ww, _, _ := pGetSystemMetrics.Call(SM_CXVIRTUALSCREEN)
	hh, _, _ := pGetSystemMetrics.Call(SM_CYVIRTUALSCREEN)
	vx, vy := int(int32(minX)), int(int32(minY))
	maxX, maxY := vx+int(ww), vy+int(hh)
	if x+w > maxX {
		x = int(at.X) - w - 12
	}
	if y+h > maxY {
		y = int(at.Y) - h - 12
	}
	if x < vx {
		x = vx
	}
	if y < vy {
		y = vy
	}
	pSetWindowPos.Call(tipHWND, HWND_TOPMOST, uintptr(x), uintptr(y), uintptr(w), uintptr(h), SWP_NOACTIVATE|SWP_SHOWWINDOW)
	pInvalidate.Call(tipHWND, 0, 1)
	tipShown = true
}

// capture: a 720x180 screen crop enlarged 1.5x for reliable small-font OCR.
// PowerPoint slides and PDF viewers often rasterize their text and need OCR.
func capture(pos point, dest string) error {
	const sourceW, sourceH = 480, 160
	const outW, outH = 960, 320
	screen, _, _ := pGetDC.Call(0)
	if screen == 0 {
		return errors.New("screen DC failed")
	}
	defer pReleaseDC.Call(0, screen)
	mem, _, _ := pCreateCompatibleDC.Call(screen)
	if mem == 0 {
		return errors.New("memory DC failed")
	}
	defer pDeleteDC.Call(mem)
	bm, _, _ := pCreateCompatibleBitmap.Call(screen, outW, outH)
	if bm == 0 {
		return errors.New("bitmap failed")
	}
	defer pDeleteObject.Call(bm)
	old, _, _ := pSelectObject.Call(mem, bm)
	pSetStretchBltMode.Call(mem, HALFTONE)
	r, _, _ := pStretchBlt.Call(mem, 0, 0, outW, outH, screen,
		uintptr(int(pos.X)-sourceW/2), uintptr(int(pos.Y)-sourceH/2),
		sourceW, sourceH, SRCCOPY)
	pSelectObject.Call(mem, old)
	if r == 0 {
		return errors.New("StretchBlt failed")
	}
	pixels := make([]byte, outW*outH*4)
	bmi := bitmapInfo{Header: bitmapInfoHeader{Size: 40, Width: outW, Height: -outH, Planes: 1, BitCount: 32, Compression: BI_RGB}}
	lines, _, _ := pGetDIBits.Call(mem, bm, 0, outH, uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&bmi)), DIB_RGB_COLORS)
	if lines == 0 {
		return errors.New("GetDIBits failed")
	}
	img := image.NewRGBA(image.Rect(0, 0, outW, outH))
	for i := 0; i < outW*outH; i++ {
		off := i * 4
		img.Pix[off] = pixels[off+2]
		img.Pix[off+1] = pixels[off+1]
		img.Pix[off+2] = pixels[off]
		img.Pix[off+3] = 255
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

type ocrWorker struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	in     io.WriteCloser
	out    *bufio.Reader
	stderr *os.File
}

func (w *ocrWorker) stop() {
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		_ = w.cmd.Wait()
	}
	w.cmd = nil
	if w.stderr != nil {
		_ = w.stderr.Close()
		w.stderr = nil
	}
}
func (w *ocrWorker) read(pos point, path string) (workerAnswer, error) {
	req := map[string]any{"sx": pos.X, "sy": pos.Y, "x": 480, "y": 160, "path": path, "warmup": path == "", "selftest": strings.EqualFold(filepath.Base(path), "test_ocr.png")}
	return w.readRequest(req)
}
func (w *ocrWorker) readSelection(pos point, path string, expectedWindow uintptr) (workerAnswer, error) {
	req := map[string]any{"sx": pos.X, "sy": pos.Y, "mode": "selection", "foregroundHwnd": expectedWindow, "path": path, "warmup": false, "selftest": false}
	return w.readRequest(req)
}
func (w *ocrWorker) readRequest(req map[string]any) (workerAnswer, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cmd == nil {
		cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-ExecutionPolicy", "Bypass", "-File", ocrPath())
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		in, err := cmd.StdinPipe()
		if err != nil {
			return workerAnswer{}, err
		}
		out, err := cmd.StdoutPipe()
		if err != nil {
			return workerAnswer{}, err
		}
		// Keep PowerShell errors, which were discarded completely in v1.1.
		if e, ee := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); ee == nil {
			w.stderr = e
			cmd.Stderr = e
		} else {
			cmd.Stderr = io.Discard
		}
		if err = cmd.Start(); err != nil {
			if w.stderr != nil {
				_ = w.stderr.Close()
				w.stderr = nil
			}
			return workerAnswer{}, err
		}
		w.cmd = cmd
		w.in = in
		w.out = bufio.NewReader(out)
	}
	raw, _ := json.Marshal(req)
	if _, err := w.in.Write(append(raw, '\n')); err != nil {
		w.stop()
		return workerAnswer{}, err
	}
	// A broken OCR subsystem must not freeze the hover scanner indefinitely.
	type readResult struct {
		line []byte
		err  error
	}
	answerCh := make(chan readResult, 1)
	go func() { line, err := w.out.ReadBytes('\n'); answerCh <- readResult{line, err} }()
	var line []byte
	var err error
	select {
	case answer := <-answerCh:
		line, err = answer.line, answer.err
	case <-time.After(10 * time.Second):
		w.stop()
		return workerAnswer{}, errors.New("OCR processing timed out")
	}
	if err != nil {
		w.stop()
		return workerAnswer{}, err
	}
	var a workerAnswer
	if err = json.Unmarshal(bytes.TrimSpace(line), &a); err != nil {
		return a, err
	}
	return a, nil
}

func analyze(pos point) hoverResult {
	out := hoverResult{P: pos}
	if ocrEngine == nil {
		// Normally prepared at startup; keep defensive lazy initialization.
		ocrEngine = &ocrWorker{}
	}
	temp := filepath.Join(os.TempDir(), fmt.Sprintf("tdd-%d.png", os.Getpid()))
	if err := capture(pos, temp); err != nil {
		out.Err = err.Error()
		return out
	}
	defer os.Remove(temp)
	a, err := ocrEngine.read(pos, temp)
	if err != nil {
		// PowerShell is sometimes blocked by corporate policy. A separately
		// installed Tesseract remains usable as an independent OCR backup.
		if w, te := nativeTesseract(temp); te == nil && w != "" {
			out.Backend = "tesseract-native"
			out.OCR = true
			out.Word = w
			// Gemini is used for word meanings too.
			return out
		}
		out.Err = "Windows OCR failed: " + err.Error() + "; Tesseract OCR also unavailable"
		return out
	}
	out.Backend = a.Backend
	out.OCR = a.OCR
	out.OCRError = a.OCRError
	out.Available = a.Available
	out.Count = a.Count
	out.Distance = a.Distance
	if !a.OK {
		out.Err = a.Error
		return out
	}
	word := strings.Trim(a.Word, " \t\r\n.,;:!?()[]{}\"‘’“”")
	if !validEnglish.MatchString(word) {
		return out
	}
	out.Word = word
	// Words are translated by Gemini; do not use stale offline/Google definitions.
	return out
}

// Independent native Tesseract fallback when corporate PowerShell restrictions
// prevent launching the Windows OCR worker. This is OPTIONAL and not bundled.
func tesseractExe() string {
	if t, e := exec.LookPath("tesseract.exe"); e == nil {
		return t
	}
	for _, p := range []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Tesseract-OCR", "tesseract.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Tesseract-OCR", "tesseract.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Tesseract-OCR", "tesseract.exe"),
	} {
		if p != "" {
			if st, e := os.Stat(p); e == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}
func nativeTesseract(path string) (string, error) {
	binary := tesseractExe()
	if binary == "" {
		return "", errors.New("Tesseract not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, path, "stdout", "-l", "eng", "--psm", "11", "tsv")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	buf, e := cmd.Output()
	if e != nil {
		return "", e
	}
	r := csv.NewReader(bytes.NewReader(buf))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	best := 1e6
	word := ""
	for {
		f, e := r.Read()
		if e == io.EOF {
			break
		}
		if e != nil || len(f) < 12 || f[0] != "5" {
			continue
		}
		conf, _ := strconv.ParseFloat(f[10], 64)
		if conf < 15 {
			continue
		}
		w := strings.Trim(f[11], " \t\r\n.,;:!?()[]{}\\\"‘’“”")
		if !validEnglish.MatchString(w) {
			continue
		}
		x, _ := strconv.ParseFloat(f[6], 64)
		y, _ := strconv.ParseFloat(f[7], 64)
		ww, _ := strconv.ParseFloat(f[8], 64)
		hh, _ := strconv.ParseFloat(f[9], 64)
		dx := math.Max(0, math.Max(x-480, 480-(x+ww)))
		dy := math.Max(0, math.Max(y-160, 160-(y+hh)))
		dist := math.Hypot(dx, dy)
		if dist < best {
			best = dist
			word = w
		}
	}
	if best <= 20 {
		return word, nil
	}
	return "", nil
}

func openDiagnosticLog() {
	_ = os.MkdirAll(dirPath(), 0755)
	if _, e := os.Stat(logPath()); errors.Is(e, os.ErrNotExist) {
		_ = os.WriteFile(logPath(), []byte("No diagnostics yet\r\n"), 0644)
	}
	pShellExecute.Call(0, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(logPath()))), 0, 0, 1)
}
func testOfflineDictionary() {
	t, ok := offlineLookup("Tachograph")
	if !ok {
		callMessage("Failed to initialize built-in dictionary", MB_ICONWARNING)
		return
	}
	callMessage("Offline dictionary OK\n\nTachograph → "+t.Meaning+"\n\nSource: "+t.Source, MB_ICONINFORMATION)
}
func test231DictionaryDisplay() {
	if tipHWND == 0 {
		callMessage("Popup window is not ready", MB_ICONWARNING)
		return
	}
	at := mouse()
	lastPos, lastChecked = at, at
	lastMoved = time.Now()
	lastCheckAt = lastMoved
	showSmall231Dictionary("Tachograph", "운행기록계", at)
	appendDiagnostic("popup display self-test invoked")
}
func openCustomDictionary() {
	prepareDictionary()
	pShellExecute.Call(0, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr("notepad.exe"))), uintptr(unsafe.Pointer(ptr(userDictionaryPath()))), 0, 1)
}
func startNetworkTest() {
	if testRunning {
		return
	}
	testRunning = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 6200*time.Millisecond)
		defer cancel()
		m, e := remoteTranslate(ctx, "hello")
		if e != nil {
			testText = "Online translation connection failed\n" + e.Error()
		} else {
			testText = "Online translation connection succeeded\nhello → " + m
		}
		appendDiagnostic("online test: " + testText)
		pPostMessage.Call(mainHWND, WM_TEST_RESULT, 0, 0)
	}()
}
func startOCRSelftest() {
	if testRunning {
		return
	}
	testRunning = true
	go func() {
		if err := extractTestImage(); err != nil {
			testText = "OCR test image creation failed: " + err.Error()
		} else {
			defer os.Remove(testImagePath())
			a, err := ocrEngine.read(point{-32000, -32000}, testImagePath())
			if (err != nil || !strings.EqualFold(a.Word, "Warranty")) && tesseractExe() != "" {
				if w, e := nativeTesseract(testImagePath()); e == nil && w != "" {
					a.Word = w
					a.Backend = "tesseract-native"
					err = nil
				}
			}
			if err != nil {
				testText = "OCR test error: " + err.Error() + "\n\nPowerShell may be restricted or English OCR language support may be missing"
			} else {
				testText = fmt.Sprintf("OCR self-test\n\nExpected: Warranty\nRecognized: %s\nEngine: %s\nOCR status: %s\nOCR error: %s\n\nIf the recognized word is Warranty, OCR is working.\nFor PPT/PDF issues, press Ctrl+Alt+Shift+D.", a.Word, a.Backend, a.Available, a.OCRError)
			}
			appendDiagnostic("Selftest: " + testText)
		}
		pPostMessage.Call(mainHWND, WM_TEST_RESULT, 0, 0)
	}()
}
func main() {
	runtime.LockOSThread()
	if len(os.Args) > 1 && os.Args[1] == "-set-gemini-key" {
		runGeminiKeyRegistrationStandalone()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "-version" {
		callMessage("231Dictionary v1.0\nCreated by Lee Sang Il\nFree for personal and business use", MB_ICONINFORMATION)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "-uninstall" {
		uninstallSelf()
		return
	}
	if !installed() {
		installSelf()
		return
	}
	prepareDictionary()
	loadSentenceConsent()
	loadBrowserExclusion()
	if err := ensureWorker(); err != nil {
		callMessage("OCR setup failed: "+err.Error(), MB_ICONWARNING)
		return
	}
	name := ptr("Local\\231Dictionary_2026_OneInstance")
	h, _, mutexErr := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	mutexHandle = h
	if h == 0 {
		callMessage("Unable to check for another running instance", MB_ICONWARNING)
		return
	}
	defer pCloseHandle.Call(mutexHandle)
	if mutexErr == syscall.Errno(183) {
		return
	}
	if err := makeWindow(); err != nil {
		callMessage(err.Error(), MB_ICONWARNING)
		return
	}
	lastMoved = time.Now()
	prepareDictionary()
	// Installer checks this only after makeWindow() has created the tray UI.
	_ = os.WriteFile(readyPath(), []byte(fmt.Sprintf("v1.0 %d %s", os.Getpid(), time.Now().Format(time.RFC3339Nano))), 0644)
	appendDiagnostic(fmt.Sprintf("v1.0 initialized; offline dictionary=%d; OCR worker starting", len(builtinDict)))
	if !geminiKeyRegistered() {
		pPostMessage.Call(mainHWND, WM_GEMINI_KEY_PROMPT, 0, 0)
	}
	// Warm up Windows OCR on startup so the first hovered word is faster.
	ocrEngine = &ocrWorker{}
	go func() {
		a, e := ocrEngine.read(point{-32000, -32000}, "")
		if e != nil {
			appendDiagnostic("OCR warmup error: " + e.Error())
		} else {
			appendDiagnostic("OCR warmup " + a.Available + "; " + a.OCRError)
		}
	}()
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	if ocrEngine != nil {
		ocrEngine.stop()
	}
	for _, obj := range []uintptr{fontWord, fontMeaning, brushBG, brushBorder} {
		if obj != 0 {
			pDeleteObject.Call(obj)
		}
	}
}
