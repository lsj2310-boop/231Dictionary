# 231Dictionary v1.1 — Windows Translator

Created by **Lee Sang Il** · Free for personal and business use.

231Dictionary is a small Windows 10/11 (64-bit) tray utility that displays Korean-English translations and key expressions in PowerPoint, selectable-text PDFs and other supported documents. Its main translation service is Google Gemini, requiring a user-supplied API key and permission for online translation.

## What's new in v1.1

**Customizable global toggle shortcut:** Use the 231Dictionary tray icon's context menu to choose **Set / Change Toggle Shortcut...**. Enter a key combination, e.g. `Ctrl+Alt+T`, and click **Save**. Pressing it in PowerPoint, a PDF viewer, or another application toggles the check mark of **Enable / Disable Word Tooltips**. The same toggle is available in the menu. The shortcut is saved for future launches.

- Accepted combinations: `Ctrl+Alt+` or `Ctrl+Shift+` plus **A–Z, 0–9, or F1–F11**. Example: `Ctrl+Shift+F5`.
- Repeated key presses do not cause rapid toggling while a key is held (`MOD_NOREPEAT`).
- If another app has registered a shortcut, the program warns you and preserves the previous shortcut. Choose another combination.
- **Remove Toggle Shortcut** clears the current shortcut. The toggle itself remains available in the tray menu.
- Changing the enabled state also follows the existing app behavior: both word tooltips and drag-selection translation are paused when disabled.
- The application starts without a global shortcut until you register one.

## Translation features

- English word tooltip: move the mouse over a word to view its meaning, when enabled and recognized.
- Drag-select English sentences to translate to Korean and view important terms underneath.
- Drag-select Korean sentences to translate to English, with an English-terms glossary in Korean.
- Maintain the source's line breaks where the document exposes selectable text.
- Exclude KakaoTalk at all times. Ordinary browser web pages may be excluded via **Disable on Web Pages (Allow PDFs)**. Browser PDFs are allowed when detected.
- Reuse the Gemini API key encrypted for your Windows user account. API and online-translation settings are not changed by the hotkey update.

## Install / update

1. Exit the running 231Dictionary from its tray menu.
2. Run **231Dictionary_Installer_v1.1.exe** and confirm installation.
3. Right-click the 231Dictionary notification-area icon.
4. Select **Set / Change Toggle Shortcut...**, type `Ctrl+Alt+T`, and choose **Save**.
5. Press `Ctrl+Alt+T` once to disable, and again to re-enable. Reopen the tray menu to inspect the check mark.
6. If needed, register a different combination or choose **Remove Toggle Shortcut**.

Existing Gemini keys, startup settings, and browser-PDF preferences are left intact. The shortcut setting is stored in `%LOCALAPPDATA%\231Dictionary\toggle_shortcut_v1.txt`.

## Security and limitations

Selected text may be transmitted to Google Gemini. Do not select confidential or personal data whose external disclosure is prohibited. Comply with company security and software installation policies. ZIP packaging does not guarantee that Windows SmartScreen will stop warning about unsigned executables.

This package was cross-compiled and unit-tested, but the installer, global shortcut registration, PowerPoint, PDF behavior, and Gemini API access were **not** directly exercised on a real Windows computer in this environment. See the verification report for scope and limitations.

The 231Dictionary About screen does not claim the MIT license; applicable third-party license notices are preserved separately as required by their terms.
