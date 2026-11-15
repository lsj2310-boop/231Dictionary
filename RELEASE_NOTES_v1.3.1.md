# 231Dictionary v1.3.1 — Clipboard reliability hotfix

Created by **Lee Sang Il** · Windows 10/11 64-bit.

## Fixes

- Clipboard-safe selection fallback for PowerPoint and supported PDF applications. It no longer restores earlier clipboard contents when the clipboard has changed again in the meantime.
- A shared Windows mutex coordinates the fallback with English popup copying to avoid two program operations overwriting one another.
- The popup right-click action copies only the completed English translation text, excluding glossary/idioms and source text. Copying retries temporary clipboard contention without blocking the user interface, and right-button capture tolerates small mouse movement.
- Guarded synthetic Ctrl+C is suppressed while keyboard modifiers or mouse buttons are held, reducing interference with ordinary copy/paste.

## Existing functionality retained

- Single-click taskbar notification icon ON/OFF, teal/gray icon state.
- Gemini English↔Korean translations, PDF Craft text selection fallback, glossary, popup left-drag movement, browser PDF allowance and KakaoTalk exclusion.

## Downloads / security

Download the EXE installer or ZIP installer. Close the previous 231Dictionary from the tray before installing. The application is not code-signed and may trigger Microsoft SmartScreen. Selected text may be sent to Google Gemini; avoid sending restricted information.

## Test scope

Go unit tests, race tests, vet checks, Windows cross-compilation and PE version inspection. Manual Windows Clipboard/PowerPoint/PDF Craft behavior was not tested in this environment.
