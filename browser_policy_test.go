package main

import "testing"

func TestBrowserExecutableAllowlist(t *testing.T) {
	browsers := []string{`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Naver\Naver Whale\Application\whale.exe`,
		`C:\Users\test\AppData\Local\Programs\Aside\Aside.exe`,
		`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
		`C:\Program Files\Mozilla Firefox\firefox.exe`,
		`C:\Program Files\Vivaldi\Application\vivaldi.exe`}
	for _, exe := range browsers {
		if !isBrowserExecutable(exe) {
			t.Errorf("browser not detected: %s", exe)
		}
	}
	applications := []string{`C:\Program Files\Microsoft Office\root\Office16\POWERPNT.EXE`,
		`C:\Program Files\Foxit Software\Foxit PDF Reader\FoxitPDFReader.exe`,
		`C:\Windows\System32\notepad.exe`, `C:\Program Files\SomeChromiumApp\SomeChromiumApp.exe`, ""}
	for _, exe := range applications {
		if isBrowserExecutable(exe) {
			t.Errorf("desktop app incorrectly excluded: %s", exe)
		}
	}
}
func TestBrowserExclusionSettings(t *testing.T) {
	for _, tc := range []struct {
		data string
		want bool
	}{
		{"", true}, {"on", true}, {"off", false}, {"false", false}, {"0", false}, {"no", false}, {"yes", true}, {"garbage", true},
	} {
		if got := browserExclusionFromFile([]byte(tc.data)); got != tc.want {
			t.Errorf("setting %q: got %v want %v", tc.data, got, tc.want)
		}
	}
}
