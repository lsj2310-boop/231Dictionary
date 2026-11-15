# 231Dictionary v1.2 — Windows Translator

**Created by Lee Sang Il** · Free for personal and business use.

231Dictionary is a Windows 10/11 (64-bit) Korean/English translation tool for PowerPoint, text-based PDFs, and other supported documents. It uses Google Gemini online translation with a personal API key.

## What's new in v1.2

- **PDF Craft text selection**: Support detection for PDF Craft Windows PDF-viewer processes, and fall back from accessibility selection to copying selectable text when applicable. Some PDF Craft editions and scanned PDFs may still be unsupported.
- **Right-click to copy English translation**: On a completed **Korean → English** translation popup, right-click to copy **only the translated English sentence(s)**, not the glossary or original text.
- **Drag popup to reposition**: Hold the left mouse button inside a translation popup and move the mouse to reposition it. It remains visible while the pointer is inside.

## Existing translation features

- English → Korean and Korean → English Gemini sentence translation.
- Show important words and expressions beneath the translation, while preserving source line breaks where available.
- Word meanings tooltips on recognized text when enabled.
- Optionally exclude regular browser web pages while allowing detected PDF documents; KakaoTalk is always excluded.
- English-language tray menu and encrypted per-user storage of the Gemini API key.

## Download and install

1. Download [231Dictionary v1.2 for Windows](https://github.com/lsj2310-boop/231Dictionary/releases/tag/v1.2). Choose the EXE directly or the ZIP installer package.
2. Exit an older running 231Dictionary from the notification-area tray icon.
3. Run `231Dictionary_Installer_v1.2.exe` and follow the installation prompts.
4. Register a personal key using **Set / Change Gemini API Key**, if none has been registered previously. Create a key in [Google AI Studio](https://aistudio.google.com/api-keys).
5. Enable **Allow Online Translation** to use Gemini-powered translations.

Existing Gemini keys and application settings are retained during the update.

## Important security notes

- Only install software approved by your company. The EXE is **not digitally code-signed** and Windows SmartScreen warnings may appear even when downloading the ZIP.
- Selected text can be transmitted to Google Gemini. Do not translate confidential company information or private data without authorization.
- Gemini API keys are not bundled with the application. Charges or rate limits depend on your Google Gemini API account and model.

## Limitations and verification

- PDF Craft refers to more than one PDF program. The fix adds recognized process names but **does not guarantee compatibility with every edition**.
- A scanned, image-only PDF may not provide selectable text. Sentence translation does not use OCR.
- The software passed automated Go tests and Windows cross-compilation. Windows UI interaction and live Gemini translation were not tested on a physical target computer.

## Build and attribution

See [BUILD.md](BUILD.md) to build from source. See [LICENSE](LICENSE) and [third-party license notice](LICENSE_MIT_MouseTooltipTranslator.txt) for distribution conditions and attribution.
