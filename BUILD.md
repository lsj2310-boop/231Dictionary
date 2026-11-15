# 231Dictionary v1.3.2 — Build / Verification

Build from the included source folder with Go 1.23+ and Python 3.

```sh
go test ./...
go test -race ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o windows_tests.exe .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w -H windowsgui' -o 231Dictionary_Installer_v1.3.2.exe .
python3 add_pe_version.py 231Dictionary_Installer_v1.3.2.exe
```

This EXE is a self-installing Windows program. No SDK or external libraries are needed to build it. Popup right-click copies the displayed translation body, synchronously, in both directions. Its clipboard transaction is coordinated with selection-copy fallback by a named mutex.

The build/test environment is Linux. Windows cross-compilation does **not** replace real PowerPoint/PDF clipboard interaction testing on a Windows desktop.
