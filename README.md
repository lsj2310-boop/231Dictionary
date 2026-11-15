# 231통역기 (231Interpreter)
Independent Android real-time bilingual speech caption app. No login, no account, no usage meter, no ads, no smart glasses.

## User interface
- Original live speech text at top and translated text at bottom.
- Default: English spoken audio → English transcript + Korean translation.
- Reverse via ⇄, other languages supported where installed speech recognizer supports them.
- Microphone start / pause, partial captions, local history, copy/share text, font size, dark/light theme, keep screen awake.
- Speech recognition engine: system default or installed system speech service; remember selected engine as the favorite.
- Local on-device ML Kit translation; first use of a language pair may download model data.

## What this does not do
- Does not modify or unlock XRAI Glass software or proprietary cloud services.
- Does not imitate XRAI's branding or graphical assets.
- Does not provide proprietary 140+ language cloud STT, AI chat assistant, speaker diarization or audio recording.
- Android SpeechRecognizer may use a remote speech service; availability, offline support, accuracy, API quotas and interruptions are device/service-dependent.
- SpeechRecognizer is segmented, so phrases are re-listened to in a loop; this is not a guaranteed always-open streaming recognition engine.

## Build
Use GitHub Actions in this branch and download artifact "231Interpreter-APK", or with Android SDK 35, Gradle 8.10.2 and Java 17 run: gradle :app:assembleDebug.
The debug APK is signed using the standard debug signing key; not suitable as a production Play Store signing credential.

## Privacy
Transcripts are kept only in the app's SQLite database, not uploaded by this app. Speech recognition may send audio to a system recognition provider; check that provider's privacy terms. Translation models are fetched by Google ML Kit, then translation operates on-device. This app does not record audio.

## Attribution
Independent implementation using Android platform speech APIs and Google ML Kit.
