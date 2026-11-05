# 231Dictionary v1.0

A Windows Korean ↔ English translation helper powered by Google Gemini.

> **Privacy:** Selected text may be sent to Google Gemini. Do not translate sensitive company information without authorization. See [SECURITY.md](SECURITY.md).

## Source and build

The included Go source builds for Windows x64 with Go 1.23 or later. See [BUILD.md](BUILD.md) for build and test steps. Installer binaries are distributed separately.

## Features and user guide

**Created by Lee Sang Il**  
Windows 10 / 11, x64. Free for personal and business use. Not an official Tata Daewoo Mobility product.

## What's new in v1.0

This release is based on **231Dictionary v3.3.1** and retains its translation features:
- The Windows tray menu, app information, installation prompts, API key dialog, and error messages are in English.
- Product identity is **231Dictionary**, **Version 1.0.0.0**, **Created by Lee Sang Il**.
- **KakaoTalk is always excluded**, whether the “Disable on Web Pages” checkbox is on or off. In KakaoTalk, neither hover-word popups nor drag-selected sentence translation should start. Other apps can still use translations.
- English-to-Korean and Korean-to-English Gemini sentence translation remains available. Translation follows the original text's line breaks and displays key words or expressions beneath it.
- Word hover support for English text is preserved from v3.3.1, including its limitations.
- Ordinary browser pages are excluded by default; PDFs inside supported browsers can still be translated.

## Install or update

1. Right-click the 231Dictionary icon in the Windows notification area and select **Exit**.
2. Run **231Dictionary_Installer_v1.0.exe**.
3. Confirm installation and select whether to start with Windows.
4. If asked, consent to online translation. Your previous Gemini API key and browser preference are preserved because the installation and settings locations are unchanged.
5. Check **About / Author / License** to confirm version **1.0** and **Lee Sang Il**.

The program installs per user under `%LOCALAPPDATA%\231Dictionary\231Dictionary.exe`. It does not require administrator privileges.

## Tray menu

| Menu item | Description |
|---|---|
| Enable / Disable Word Tooltips | Toggle translations on/off |
| Start with Windows | Autostart when signing in |
| Help and Status | View available features and current settings |
| About / Author / License | App name, v1.0, creator, usage conditions |
| Set / Change Gemini API Key | Store the key encrypted with Windows DPAPI |
| Disable on Web Pages (Allow PDFs) | Exclude ordinary browser pages but permit PDFs inside browsers |
| Allow Online Translation | Opt in or opt out of sending selected text to Gemini |
| Uninstall 231Dictionary | Uninstall this app |
| Exit | Close the running app |

**KakaoTalk is excluded unconditionally** and therefore has no additional checkbox.

## Usage examples

Select English text with a newline between: `Hand pump works.` and `Electric pump fails to build hydraulic pressure.`
The popup should show Korean translations on the corresponding lines, followed by key expressions such as `hand pump` and `hydraulic pressure` with Korean meanings.

Select Korean text with a newline between: `전동 펌프가 작동하지 않습니다.` and `유압이 부족합니다.`
The popup should show English translations on corresponding lines, followed by key English terms and their Korean meanings.

In PowerPoint, select editable text inside its text box or table. In Foxit/Adobe PDF, use the text-selection tool. Scanned-image PDF content cannot be selected through the text-selection flow.

## Known limitations

- Live Windows PowerPoint, Foxit PDF, KakaoTalk, browser and Gemini tests **could not be performed** in the Linux build environment; static logic/unit tests and a Windows x64 build were performed instead.
- Some browser PDF viewers may not expose PDF status to Windows UI Automation, so browser-PDF identification may fail.
- For applications without selectable text accessibility, the copy-based selection fallback may not work. It attempts to restore the clipboard, but restoration may not be perfect for every format.
- The program may send selected sentences or words to Google's Gemini service. Do not translate confidential data unless company policy permits it.
- Gemini access requires a working API key, model availability and network permission. A 404, 429 or 503 response may indicate model access, quota or temporary overload respectively.
- The existing word-hover feature inherited from v3.3.1 may be unreliable in some PowerPoint/PDF setups. This release does not claim to repair it.

## Troubleshooting

If a popup appears in KakaoTalk, verify that the active executable is really `KakaoTalk.exe`, close any older 231Dictionary instances, then reopen v1.0. For translation failures, check `%LOCALAPPDATA%\231Dictionary\diagnostic.log`. The privacy-sensitive selected sentence is not stored in this log by the translation flow.

## Distribution and credits

The original application author is **Lee Sang Il**; this app is distributed free for personal and business use, without warranty. See `LICENSE` for the program's usage statement. The third-party source that was adapted carries its own original notice under `LICENSE_MIT_MouseTooltipTranslator.txt`. Such third-party notices do not change the program name or version.
