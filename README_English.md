# 231Dictionary v1.2 — Windows Korean/English Translator

Created by **Lee Sang Il**. Free for personal and business use.

## What's improved in v1.2

- **PDF Craft compatibility:** selecting text in the PDF Craft Windows application now uses the existing Windows accessibility text-selection path and, when necessary, a carefully scoped copy-text fallback. PDF Craft executable names (`PDFCraft.exe`, `PDFCraftPro.exe`, and compatible editions) are recognized. Text selection must be enabled, and scanned image-only PDF content cannot be translated by direct-text extraction.
- **Copy English translation:** on a completed Korean-to-English sentence popup, **right-click** to copy only the translated English sentence(s). The English vocabulary/glossary below it is *not* included. Right-clicking the popup before translation completes does not overwrite the clipboard.
- **Move translation popup:** hold the **left mouse button** inside the popup, drag it to another position, and release. The popup stays open while the mouse is inside it and closes after the mouse leaves it.

## Existing features

- English-to-Korean and Korean-to-English translations powered by Google Gemini (a personal API key is required).
- Highlight or drag-select text in supported PowerPoint and PDF applications, and retain source line breaks in the translated text where possible.
- Show important terms and expressions under sentence translations.
- Hover translation where recognized and enabled.
- Option to disable translation on ordinary browser web pages while permitting browser PDF readers.
- Always disabled in KakaoTalk.
- Configure a global keyboard shortcut for enabling/disabling translation from the tray menu.

## Installation

1. Exit an existing 231Dictionary instance using the notification-area tray menu.
2. Run `231Dictionary_Installer_v1.2.exe` (or unzip `231Dictionary_v1.2_Installer.zip` and run the included EXE).
3. In Windows notification area, right-click 231Dictionary; choose **Set / Change Gemini API Key** if a key has not previously been registered.
4. Make sure **Allow Online Translation** is enabled.
5. In PDF Craft, select **Select text** (not annotation or hand tool). Drag to highlight actual PDF text, then release.
6. On a Korean-to-English translated sentence, right-click the popup to copy the English translation, or left-drag the popup to reposition it.

## Important security notes

Selected text is sent to Google Gemini. Never send company-confidential information or personally identifiable data unless your organization expressly permits it.

The installer is not digitally code-signed. Windows SmartScreen may warn even when the program is distributed inside a ZIP. Obey the software-installation rules of your organization.

Previous Gemini API credentials and application settings are retained. No API keys are included in the distributed source or executable.

## Limitations

- PDF Craft is the name of more than one distinct PDF program. Exact process names and text-selection implementations vary. Direct text access has been extended for common PDF Craft process names, but actual operation **has not been verified on the user's Windows PC**.
- Selected text cannot be extracted from image-only PDFs without OCR, which the sentence selection workflow deliberately does not use.
- This source was cross-compiled for Windows and unit-tested in a non-Windows environment; Windows GUI mouse interactions and the live Gemini service require an on-device check.
- The right-click copy function is **only** for completed Korean-to-English sentence translations; it does not copy the original Korean text or any glossary entries.

Third-party notices are included separately in `LICENSE_MIT_MouseTooltipTranslator.txt`.
