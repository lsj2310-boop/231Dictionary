# 231Dictionary v1.3.2 — Translation Popup Copy Correction

Created by **Lee Sang Il** · Windows 10/11 (64-bit).

## What was corrected

- Right-clicking a completed sentence translation popup now copies **the displayed translation text**, not the selected source text from PowerPoint or a PDF viewer.
- Both translation directions are supported: **English → Korean** and **Korean → English**.
- The copied text excludes the original sentence and **Key Words & Expressions** glossary.
- Clipboard writing completes before the right-click handler returns, so immediately pasting cannot accidentally use older source text because of a delayed copy.
- A temporary clipboard-lock failure shows an explicit error rather than silently leaving the old clipboard content.
- No copy is attempted while translation is pending, when an error is shown, or for a word-tooltip popup.

## Unchanged

- Teal/gray tray icon with single-click translation toggle.
- PDF Craft text selection support, popup left-drag movement, Gemini API key storage, browser-PDF exception and KakaoTalk exclusion.
- Clipboard preservation safeguards for normal PowerPoint/PDF copy and paste.

## Install / test

Close the older version using **Exit** in the tray icon menu; run `231Dictionary_Installer_v1.3.2.exe` (or unpack the ZIP). Your existing Gemini API key is preserved. Translate an English sentence, then right-click the Korean translation and paste it into Notepad: only Korean should be pasted. Repeat for Korean → English; only the English translation should be pasted.

## Security and verification

The EXE is unsigned and may cause a Windows SmartScreen warning. Translated selection content is sent to Google Gemini; follow your organization's data handling policies. Static Go tests, race checks, vet and Windows cross-compilation are performed, but the mouse/clipboard behavior has **not** been manually exercised on a real Windows desktop in this environment.
