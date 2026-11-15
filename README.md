# 231통역기 v1.1 — Gemini Live

Native Android bilingual real-time speech transcription and translation app. Independent app, no login or AI glasses.

## v1.1 changes
- Default microphone engine: Gemini 3.5 Transcribe Live via official Google Live API WebSocket. Speech 16 kHz PCM mono, streamed every 100 ms after setupComplete. Shows interim recognition immediately, commits final utterances without restarting Android SpeechRecognizer.
- Final text translated via Gemini 3.5 Flash-Lite. Provisional text uses on-device ML Kit for speed after model installation. If ML Kit model is not ready, interim text uses Gemini API.
- In-app Gemini API key registration and replacement. Android Keystore AES-GCM encryption at rest; no key embedded in APK. Can clear key in Settings. No user login.
- Android system speech engine remains available when Gemini mode is turned off in Settings.
- English original at top / Korean translation at bottom by default. Can reverse language direction and adjust size, theme, history, share.
- The Google API requires a key and internet; quota, availability, prices and model access are controlled by Google.
- BYOK security note: for corporate/production use, a small backend issuing ephemeral Gemini tokens is more secure than storing a long-lived key on a device. Audio is sent to Google for remote speech recognition; use only where permitted.

## Build
Java 17, Gradle 8.10.2, Android SDK 35.
GitHub Actions releases APK 231Interpreter-v1.1.apk under tag 231-interpreter-v1.1.
Debug-signed APK for test sideloading, not Play Store.
