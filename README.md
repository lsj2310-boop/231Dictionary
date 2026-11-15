# 231Dictionary v1.3.2 — Windows Korean/English Translator

Created by **Lee Sang Il** · Free for personal and business use.

231Dictionary is a Windows 10/11 (64-bit) notification-area translation utility for PowerPoint, selectable-text PDF documents (including supported PDF Craft views), and other supported programs. Online translation is provided by Google Gemini and requires your own Gemini API key.

## Correction in v1.3.2 — Copy the displayed translation

- Right-click a completed sentence translation popup to copy **only the translated sentence currently displayed**.
- Supports English → Korean and Korean → English equally, and excludes the source sentence and glossary.
- The clipboard operation is completed before right-click processing returns; a clipboard error is shown rather than silently copying nothing.
- Left-click and drag the popup to reposition it. Existing single-click tray toggle and clipboard preservation are retained.

## From v1.3 — Single-click ON/OFF tray icon

**Left-click the 231Dictionary icon once in the Windows notification area** (the system tray, beside the clock; open `^` if hidden):

- **Teal / colored icon:** translation is **ON**. Word tooltips and drag-selection translation are enabled.
- **Gray icon:** translation is **OFF**. Both word tooltips and sentence translation are paused.
- Click it again to restore translation. The icon and its hover description update immediately.
- Your ON/OFF preference is remembered when you exit and relaunch the program.
- **Right-click** still opens the options menu. The checked status of **Enable / Disable Word Tooltips** remains synchronized with the icon. The existing options menu uses the same toggle.

The toggle does not remove your encrypted Gemini API key, change online-translation consent, or change browser exclusions.

## Existing functions retained

- Hover over an English word for its meaning when supported.
- Drag-select an English sentence to translate it into Korean; view important English words and expressions underneath.
- Drag-select a Korean sentence to translate it into English; view key English expressions with Korean explanations.
- Preserve source line breaks when text selection provides them.
- Right-click a **sentence translation popup** to copy the *currently displayed Korean or English translation only*, without the source sentence or glossary.
- Left-click and drag the popup to move it; keep the popup visible while the mouse is inside it.
- Exclude KakaoTalk. Optionally disable ordinary browser pages while allowing recognized browser PDFs.
- Attempt PDF Craft text selection by its program-aware selection/clipboard path.

## Installation / updating

1. Exit the running 231Dictionary from its notification-area menu (**Exit**).
2. Run `231Dictionary_Installer_v1.3.2.exe`, or extract `231Dictionary_v1.3.2_Installer.zip` and run the EXE inside.
3. Restart 231Dictionary; its notification-area icon shows the current translation state (teal = ON; gray = OFF).
4. Left-click once to toggle. Right-click for the options menu.
5. For first-time use, choose **Set / Change Gemini API Key** from the right-click menu and register your own Google AI Studio API key. Ensure **Allow Online Translation** is enabled.

The toggle state is saved at `%LOCALAPPDATA%\231Dictionary\translation_enabled_v1.txt`. Existing Gemini API credentials and other preferences are preserved.

## Security and limits

Selected text may be sent to Google Gemini. Do not translate confidential, personal, or restricted information without organizational authorization. Comply with your organization's software installation policy.

Unsigned Windows EXEs can trigger Microsoft Defender SmartScreen even when distributed in ZIP files. ZIP packaging is not a security certification.

The Windows GUI and tray callbacks cannot be executed directly in the Linux build environment. Go unit tests and Windows x64 cross-compilation passed; **test real tray click, color change, Explorer behavior, and translation on a Windows machine before wider distribution**.

Third-party attribution is in `LICENSE_MIT_MouseTooltipTranslator.txt`; the About dialog does not claim MIT for the program itself.
