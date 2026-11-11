# 231Dictionary v1.1 — Windows Translator

**Created by Lee Sang Il** · Free for personal and business use.

231Dictionary is a Windows 10/11 (64-bit) translation utility for English and Korean text. It helps users read PowerPoint presentations, selectable-text PDF files, and other supported documents without repeatedly copying text into a separate translation site.

## Translation features

- **English → Korean:** Select an English sentence with the mouse to translate it into Korean and see important English terms and expressions explained underneath.
- **Korean → English:** Select a Korean sentence to translate it into English, with key English expressions explained in Korean.
- **Word meanings:** When word recognition is supported, moving the mouse over an English word can show a small Korean meaning tooltip.
- **Line breaks:** Preserve source line breaks where the selected text provides them.
- **App exclusions:** Always exclude KakaoTalk. Regular browser web pages may be excluded via **Disable on Web Pages (Allow PDFs)**, while detectable browser PDF documents remain supported.
- **Gemini integration:** A personal Google Gemini API key is required for online translation and is stored encrypted for the current Windows user.

## Installation and use

1. Visit the [v1.1 Releases page](https://github.com/lsj2310-boop/231Dictionary/releases/tag/v1.1).
2. Download `231Dictionary_Installer_v1.1.exe`, or download `231Dictionary_v1.1_Installer.zip` and extract its contents.
3. Exit any running 231Dictionary instance from the Windows notification area before installing the update.
4. Run the installer and start 231Dictionary.
5. Right-click the notification-area icon. Choose **Set / Change Gemini API Key** to register your own API key from [Google AI Studio](https://aistudio.google.com/api-keys).
6. Ensure **Allow Online Translation** is enabled and select English or Korean text in a supported document.

Existing Gemini API credentials and settings are preserved when updating from earlier versions.

## Supported environments and limitations

- Windows 10/11 (64-bit).
- Text recognition depends on the document viewer and whether the text can be selected or exposed through accessibility interfaces.
- Translation needs an internet connection and access to the Google Gemini API. Google may apply quotas or charges depending on the model and account.
- Some PDF viewers, presentations, or network environments may limit functionality.

## Security and licensing

Selected text may be sent to Google Gemini for translation. Do not send confidential, personal, or restricted business information without authorization. Follow your organization's software installation and data handling policies.

The installer is not digitally code-signed; downloading it in a ZIP archive does **not** guarantee Windows SmartScreen warnings will disappear.

See `LICENSE` for the application's distribution terms and `LICENSE_MIT_MouseTooltipTranslator.txt` for third-party attribution. The About screen does not claim that the application itself is distributed under the MIT license.

## Building from source

See [BUILD.md](BUILD.md) for Windows build instructions.
