# 231Dictionary v1.0 Build and Verification

Based on 231Dictionary v3.3.1 source, preserving its Gemini, selected-text, glossary and browser-PDF behavior.

- English local user interface, tray menu, installer dialogs, status, version information, key prompt and error messages.
- Developer: Lee Sang Il (previously Korean display). Product: 231Dictionary v1.0.0.0.
- New executable-name-based KakaoTalk exclusion, always active independently of the optional browser exclusion setting. Checks both foreground and pointer windows. Checks are repeated when asynchronous translation results arrive and when popups are about to be displayed.
- Browser PDF remains allowed; ordinary web page suppression remains configurable.
- User's existing DPAPI key and browser settings paths preserved.
- The original third-party license notice is retained in the distribution but not placed on the About screen.

Run `go test ./...`, `go test -race ./...`, `go vet ./...`; cross-build with `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -H windowsgui' -o 231Dictionary.exe .`; attach PE VERSIONINFO via `add_pe_version.py`.

Tests can verify classification, source integration, build, and packages. Actual Windows application suppression and network translation require live verification on the target machine.
