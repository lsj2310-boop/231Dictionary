package main

import (
	"regexp"
	"sort"
	"strings"
)

var wordToken = regexp.MustCompile(`[A-Za-z]+(?:['’-][A-Za-z]+)*`)
var fixedPhrases = map[string]string{
	"root cause": "근본 원인", "root cause analysis": "근본 원인 분석",
	"quality assurance": "품질 보증", "customer satisfaction": "고객 만족",
	"corrective action": "시정 조치", "preventive action": "예방 조치",
	"interim containment action": "임시 확산 방지 조치", "permanent corrective action": "영구 시정 조치",
	"software update": "소프트웨어 업데이트", "field fix": "현장 개선 조치",
	"warranty claim": "보증 청구", "warranty period": "보증기간",
	"service campaign": "서비스 캠페인", "recall campaign": "리콜 캠페인",
	"failure mode": "고장 형태", "failure analysis": "고장 분석",
	"quality issue": "품질 문제", "vehicle inspection": "차량 검사",
	"in accordance with": "~에 따라", "as a result": "그 결과",
	"due to": "~때문에", "in order to": "~하기 위해",
	"carry out": "수행하다", "follow up": "후속 조치하다",
	"break down": "고장 나다", "take place": "발생하다",
	"be responsible for": "~을 담당하다", "take action": "조치하다",
	"set up": "설정하다/설치하다", "look into": "조사하다",
	"work out": "해결하다", "point out": "지적하다",
	"risk assessment": "위험 평가", "design change": "설계 변경",
	"service life": "수명", "production line": "생산 라인",
	"cab tilt system": "캡 틸트 시스템", "hand pump": "수동 펌프", "electric pump": "전동 펌프",
	"hydraulic pressure": "유압", "condenser motor": "콘덴서 모터", "condenser motor failure": "콘덴서 모터 고장",
	"air conditioning": "에어컨, 공조", "a c cooling": "에어컨 냉방", "insufficient cooling": "냉방 성능 부족",
	"engine mounting": "엔진 마운팅", "ignition key": "시동 키", "vehicle battery": "차량 배터리",
	"issue summary": "이슈 요약", "customer complaint": "고객 불만",
	"leak test": "누설 시험", "water leakage": "누수",
	"target vehicle": "대상 차량", "implementation date": "적용일",
	"effective date": "시행일", "action plan": "조치 계획",
	"software version": "소프트웨어 버전", "vehicle speed": "차량 속도",
	"road test": "주행 시험", "performance evaluation": "성능 평가",
}
var stopWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields("the a an is are was were be been being to of at in on for with from by as and or but that this these those it its we you they them our their your there here has have had do does did will would shall should can could may might must not no if then than which who what when where how while after before during into out up down about also very more most some any each all both per via new first last next same different such use using used any") {
		stopWords[w] = true
	}
}
func extractKeyTerms(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	lower := " " + strings.ToLower(strings.Join(wordToken.FindAllString(s, -1), " ")) + " "
	// Match long phrases first; only meaningful combinations.
	type pair struct{ k, v string }
	items := make([]pair, 0, limit)
	used := make(map[string]bool)
	// Prefer the longest specific phrases and avoid listing nested duplicates.
	keys := make([]string, 0, len(fixedPhrases))
	for phrase := range fixedPhrases {
		keys = append(keys, phrase)
	}
	sort.Slice(keys, func(i, j int) bool {
		wi, wj := len(strings.Fields(keys[i])), len(strings.Fields(keys[j]))
		if wi != wj {
			return wi > wj
		}
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		if len(items) >= limit {
			break
		}
		if !strings.Contains(lower, " "+k+" ") {
			continue
		}
		contained := false
		for phrase := range used {
			if strings.Contains(" "+phrase+" ", " "+k+" ") {
				contained = true
				break
			}
		}
		if !contained {
			items = append(items, pair{k, fixedPhrases[k]})
			used[k] = true
		}
	}
	for _, w := range wordToken.FindAllString(s, -1) {
		if len(items) >= limit {
			break
		}
		k := strings.ToLower(w)
		if stopWords[k] || used[k] || len(k) < 4 {
			continue
		}
		if v, ok := offlineLookup(w); ok {
			// Avoid repeatedly displaying words already explained by a recognized phrase.
			covered := false
			for p := range used {
				if strings.Contains(" "+p+" ", " "+k+" ") {
					covered = true
					break
				}
			}
			if covered {
				continue
			}
			meaning := []rune(v.Meaning)
			if len(meaning) > 30 {
				meaning = meaning[:30]
			}
			items = append(items, pair{w, string(meaning)})
			used[k] = true
		}
	}
	var lines []string
	for _, p := range items {
		lines = append(lines, p.k+"  ·  "+p.v)
	}
	return strings.Join(lines, "\n")
}
