from pathlib import Path
root=Path(__file__).parent
p=root/'main.go';s=p.read_text()
def rep(old,new,count=1):
 global s
 n=s.count(old)
 if n!=count: raise RuntimeError(f'Expected {count} occurrences, got {n}: {old[:90]}')
 s=s.replace(old,new)
rep('//go:embed ocr_worker.ps1 test_ocr.png web_proxy.ps1 browser_pdf_probe.ps1 LICENSE_MIT_MouseTooltipTranslator.txt','//go:embed ocr_worker.ps1 test_ocr.png web_proxy.ps1 browser_pdf_probe.ps1 LICENSE_MIT_MouseTooltipTranslator.txt tray_on.bin tray_off.bin')
rep('NIM_ADD              = 0x0\n\tNIM_DELETE           = 0x2','NIM_ADD              = 0x0\n\tNIM_MODIFY           = 0x1\n\tNIM_DELETE           = 0x2')
rep('icon, _, _ := pLoadIcon.Call(0, uintptr(32516)) // IDI_INFORMATION','initializeTrayIcons()\n\ticon := selectedTrayIcon() // Colored when enabled; gray when paused')
rep('addTray(icon)\n\tpSetTimer.Call','addTray()\n\tpSetTimer.Call')
a=s.index('func addTray(icon uintptr) {');b=s.index('\nfunc cleanupTray()',a)
s=s[:a]+'''func addTray() {
\tvar ni notifyIconData
\tfillTrayNotification(&ni, NIF_MESSAGE|NIF_ICON|NIF_TIP)
\tni.CallbackMessage = WM_TRAY
\tif ok, _, err := pShellNotify.Call(NIM_ADD, uintptr(unsafe.Pointer(&ni))); ok == 0 {
\t\tappendDiagnostic("Shell_NotifyIcon(NIM_ADD) failed: " + fmt.Sprint(err))
\t}
}'''+s[b:]
rep('''\t\tcase WM_TRAY:
\t\t\tif lp == WM_RBUTTONUP {
\t\t\t\tmenu()
\t\t\t} else if lp == WM_LBUTTONDBLCLK {
\t\t\t\tshowInfo()
\t\t\t}
\t\t\treturn 0''','''\t\tcase WM_TRAY:
\t\t\tswitch lp {
\t\t\tcase WM_RBUTTONUP:
\t\t\t\tmenu()
\t\t\tcase WM_LBUTTONUP:
\t\t\t\t// The second mouse-up in a double click must not undo the first toggle.
\t\t\t\tif time.Now().UnixNano() >= trayDoubleClickUntil {
\t\t\t\t\ttoggleTooltipsFromMenuOrShortcut()
\t\t\t\t}
\t\t\tcase WM_LBUTTONDBLCLK:
\t\t\t\ttrayDoubleClickUntil = time.Now().Add(800 * time.Millisecond).UnixNano()
\t\t\t}
\t\t\treturn 0''')
rep('loadBrowserExclusion()\n\tif err := ensureWorker();','loadBrowserExclusion()\n\tloadTranslationState() // Restore last ON/OFF state before adding tray icon.\n\tif err := ensureWorker();')
rep('if ocrEngine != nil {\n\t\tocrEngine.stop()\n\t}\n\tfor _, obj := range','if ocrEngine != nil {\n\t\tocrEngine.stop()\n\t}\n\tdisposeTrayIcons()\n\tfor _, obj := range')
s=s.replace('231Dictionary v1.2','231Dictionary v1.3').replace('v1.2 initialized','v1.3 initialized').replace('v1.2 %d %s','v1.3 %d %s')
p.write_text(s)
p=root/'toggle_shortcut_windows.go';s=p.read_text()
old='appendDiagnostic("Tooltips toggled " + state)'
assert s.count(old)==1
s=s.replace(old,'saveTranslationState()\n\trefreshTrayIcon() // Updates status color for tray, shortcut, and menu toggles.\n\t'+old)
p.write_text(s)
p=root/'add_pe_version.py';s=p.read_text().replace('major, minor, patch, build = 1, 2, 0, 0','major, minor, patch, build = 1, 3, 0, 0').replace('1.2.0.0','1.3.0.0');p.write_text(s)
print('all changes applied')
