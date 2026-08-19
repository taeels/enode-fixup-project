package schema

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustSchema(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// ★ ADR-020 의 경계선 ★
// 기준: 에이전트가 정직하게 답했을 때 통과하지 못할 수 있으면 그건 판정이다.
func TestBoundary(t *testing.T) {
	ok := []string{
		`{"type":"object","required":["status"],"properties":{
		    "status":{"enum":["found","none"]},
		    "hypothesis":{"type":"string"},"reason":{"type":"string"}}}`,
		`{"type":"array","items":{"type":"string"}}`,
		`{"type":"object","additionalProperties":false,"properties":{"a":{"type":"string"}}}`,
	}
	for _, s := range ok {
		if err := CheckBoundary(mustSchema(t, s)); err != nil {
			t.Errorf("형식만 쓴 스키마가 거절됐다: %v", err)
		}
	}

	bad := map[string]string{
		`{"type":"number","minimum":0.8}`:                                      "minimum",
		`{"type":"string","minLength":200}`:                                    "minLength",
		`{"type":"string","pattern":"^x"}`:                                     "pattern",
		`{"type":"array","maxItems":3}`:                                        "maxItems",
		`{"type":"object","properties":{"c":{"type":"number","minimum":0.8}}}`: "minimum",
		`{"type":"array","items":{"type":"string","minLength":10}}`:            "minLength",
		`{"type":"string","format":"email"}`:                                   "format",
	}
	for s, kw := range bad {
		err := CheckBoundary(mustSchema(t, s))
		if err == nil {
			t.Fatalf("★ 판정 키워드가 통과했다 ★ %s — ADR-004 가 스키마를 통해 새어나간다", kw)
		}
		if !strings.Contains(err.Error(), kw) {
			t.Errorf("어느 키워드가 문제인지가 안 나온다: %v", err)
		}
	}
}

func TestValidate(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","required":["status"],"properties":{
	  "status":{"enum":["found","none"]},
	  "hypothesis":{"type":"string"},"reason":{"type":"string"}}}`)

	// ★ 에이전트가 정직하게 "못 하겠다" 를 말할 수 있다 ★ (ADR-020)
	// 부재로 표현하면 크래시와 구분이 안 된다.
	for _, doc := range []string{
		`{"status":"found","hypothesis":"락 순서가 …"}`,
		`{"status":"none","reason":"패치가 주석만 바꾼다"}`,
	} {
		if v := Validate(sch, []byte(doc)); len(v) != 0 {
			t.Errorf("정직한 답이 거절됐다: %s → %v", doc, v)
		}
	}

	bad := map[string]string{
		`{"hypothesis":"x"}`:                "status", // 필수가 없다
		`{"status":"maybe"}`:                "enum",   // 어휘 밖
		`{"status":"found","hypothesis":3}`: "타입",     // 타입이 틀렸다
		`not json`:                          "JSON",
	}
	for doc, want := range bad {
		v := Validate(sch, []byte(doc))
		if len(v) == 0 {
			t.Fatalf("★ 어긴 문서가 통과했다 ★: %s", doc)
		}
		joined := ""
		for _, x := range v {
			joined += x.String() + " "
		}
		if !strings.Contains(joined, want) {
			t.Errorf("%s → %q 에 %q 가 없다", doc, joined, want)
		}
	}
}

// 위반 내역은 ★ feedback 으로 되먹여진다 ★ — 무엇이 왜 틀렸는지가 있어야 한다.
func TestViolationIsActionable(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","required":["status"]}`)
	v := Validate(sch, []byte(`{}`))
	if len(v) != 1 || v[0].Path != "status" || v[0].Got != "없음" {
		t.Fatalf("위반 내역이 부실하다: %+v", v)
	}
	if !strings.Contains(v[0].String(), "status") {
		t.Fatalf("사람이 읽을 수 없다: %s", v[0])
	}
}
