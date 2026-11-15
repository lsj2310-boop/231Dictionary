# 231Dictionary v1.2 — Build and Verification

Windows 10/11 x64 GUI translation utility.

## Build prerequisites

- Go 1.23.2 or compatible Go 1.23+ release
- Python 3 for PE version information
- Go project has standard-library-only dependencies

```bash
go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w -H windowsgui' -o 231Dictionary_Installer_v1.2.exe .
python3 add_pe_version.py 231Dictionary_Installer_v1.2.exe
```

Expected SHA-256 of the tested v1.2 EXE:

`11bbd89e10c3027359a8d9b87d1e53d0c498eac0a5bafa6c14c65876a257a800`

Do not distribute or test with real confidential customer/company records via public Gemini APIs unless your organization authorizes such transfer. The source includes PDF Craft selection handling, right-click English copy, and mouse-drag popup positioning, but running Windows GUI/PDF Craft tests requires a Windows computer.
