# Security and privacy

231Dictionary sends selected text to Google Gemini only when online translation is enabled and the user has configured their own API key. Do not use it on confidential or proprietary content without appropriate authorization. Please consult your organization's security rules before running third-party software.

API keys are not included in this repository. Do not post API keys in issues, pull requests, code, screenshots, or logs. On Windows, the application stores the configured key using DPAPI. If a key is exposed, revoke and rotate it in Google AI Studio.

The program can temporarily access the clipboard to retrieve selected text when accessibility APIs cannot provide it. Clipboard restoration is best-effort.

The source has not been independently audited, and actual PowerPoint/PDF/KakaoTalk integration behavior requires Windows testing. Report security concerns without including secret credentials or confidential content.
