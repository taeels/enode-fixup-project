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

// ADR-020 의 경계선
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
			t.Errorf("a shape-only schema was rejected: %v", err)
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
			t.Fatalf("a verdict keyword passed: %s — ADR-004 leaks through the schema", kw)
		}
		if !strings.Contains(err.Error(), kw) {
			t.Errorf("the offending keyword is not named: %v", err)
		}
	}
}

func TestValidate(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","required":["status"],"properties":{
	  "status":{"enum":["found","none"]},
	  "hypothesis":{"type":"string"},"reason":{"type":"string"}}}`)

	// 에이전트가 정직하게 "못 하겠다" 를 말할 수 있다 (ADR-020)
	// 부재로 표현하면 크래시와 구분이 안 된다.
	for _, doc := range []string{
		`{"status":"found","hypothesis":"lock ordering …"}`,
		`{"status":"none","reason":"the patch only changes comments"}`,
	} {
		if v := Validate(sch, []byte(doc)); len(v) != 0 {
			t.Errorf("an honest answer was rejected: %s → %v", doc, v)
		}
	}

	bad := map[string]string{
		`{"hypothesis":"x"}`:                "status",        // required is missing
		`{"status":"maybe"}`:                "enum",          // outside the vocabulary
		`{"status":"found","hypothesis":3}`: "expected type", // wrong type
		`not json`:                          "JSON",
	}
	for doc, want := range bad {
		v := Validate(sch, []byte(doc))
		if len(v) == 0 {
			t.Fatalf("a violating document passed: %s", doc)
		}
		joined := ""
		for _, x := range v {
			joined += x.String() + " "
		}
		if !strings.Contains(joined, want) {
			t.Errorf("%s → %q does not contain %q", doc, joined, want)
		}
	}
}

// 위반 내역은 feedback 으로 되먹여진다 — 무엇이 왜 틀렸는지가 있어야 한다.
func TestViolationIsActionable(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","required":["status"]}`)
	v := Validate(sch, []byte(`{}`))
	if len(v) != 1 || v[0].Path != "status" || v[0].Got != "missing" {
		t.Fatalf("the violation detail is too thin: %+v", v)
	}
	if !strings.Contains(v[0].String(), "status") {
		t.Fatalf("not readable by a person: %s", v[0])
	}
}
