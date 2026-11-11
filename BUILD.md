# 231Dictionary v1.1 — Build / Verification

Windows 10/11 x64 GUI program, built from the v1.0 English/KakaoTalk-exclusion codebase. New functionality: persistent global shortcut to toggle the existing **Enable / Disable Word Tooltips** action.

## Build prerequisites

Go 1.23+ and Python 3. The Go project uses only Go standard-library dependencies.

```bash
go test ./...
go test -race ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -H windowsgui' -o 231Dictionary_Installer_v1.1.exe .
python3 add_pe_version.py 231Dictionary_Installer_v1.1.exe
```

`RegisterHotKey` (Win32) registers a non-repeating hotkey with the app's hidden main window; `WM_HOTKEY` and the existing tray command invoke the same toggle function. A separate native Win32 settings window accepts combinations such as `Ctrl+Alt+T` and `Ctrl+Shift+F5` and saves them to `%LOCALAPPDATA%\231Dictionary\toggle_shortcut_v1.txt`. Shortcut changes are only persisted after a successful registration. Invalid or occupied shortcuts do not displace the existing shortcut. Close/uninstall unregisters the global hotkey; the user can also remove it via the tray menu.

This build retains translation, glossary, browser-PDF, KakaoTalk-exclusion, Gemini encryption, and English-language UI. The About UI does not show "MIT"; applicable third-party notices remain with the software.

Cross-compilation and unit tests alone do **not** establish that the full tray/dialog behavior or Gemini requests work on a real Windows PC. Test the menu checkbox, shortcut persistence after restart, conflicts, and PowerPoint/PDF behavior on a target computer before enterprise deployment.
