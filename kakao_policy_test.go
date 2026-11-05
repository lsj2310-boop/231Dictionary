package main

import "testing"

func TestKakaoTalkAlwaysExcluded(t *testing.T) {
	cases := []string{"KakaoTalk.exe", `C:\\Program Files (x86)\\Kakao\\KakaoTalk\\KakaoTalk.exe`, "KAKAOTALK.EXE", "KakaoTalkBeta.exe", "KakaoTalkPC.exe"}
	for _, name := range cases {
		for _, excludeBrowsers := range []bool{true, false} {
			for _, pdf := range []bool{true, false} {
				if !shouldSuppressApplication(name, excludeBrowsers, pdf) {
					t.Errorf("KakaoTalk should always be excluded: %q excludeBrowsers=%v pdf=%v", name, excludeBrowsers, pdf)
				}
			}
		}
	}
}
func TestBrowserPDFAllowedAndWebpageOptional(t *testing.T) {
	for _, browser := range []string{"chrome.exe", "whale.exe", "brave.exe", "msedge.exe", "aside.exe"} {
		if !shouldSuppressApplication(browser, true, false) {
			t.Errorf("browser webpage not excluded: %s", browser)
		}
		if shouldSuppressApplication(browser, true, true) {
			t.Errorf("browser PDF blocked: %s", browser)
		}
		if shouldSuppressApplication(browser, false, false) {
			t.Errorf("browser not reenabled: %s", browser)
		}
	}
	for _, app := range []string{"powerpnt.exe", "foxitpdfreader.exe", "winword.exe", "notepad.exe"} {
		if shouldSuppressApplication(app, true, false) {
			t.Errorf("nonbrowser excluded: %s", app)
		}
	}
}
func TestUnrelatedKakaoNamesNotExcluded(t *testing.T) {
	for _, app := range []string{"", "kakao.exe", "KakaoTalkBackup.exe", "kakaotalk.exe.txt", "firefox.exe"} {
		if isKakaoTalkExecutable(app) {
			t.Errorf("false match for %q", app)
		}
	}
}
