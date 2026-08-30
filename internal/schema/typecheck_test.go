package schema

import (
	"math"
	"strings"
	"testing"
)

// 타입 위반은 「어느 자리가 · 무엇을 기대했고 · 무엇이 왔는가」 셋을 다 말한다.
//
// 이 셋이 feedback 으로 되먹여진다 (ADR-013 의 루프). 하나라도 빠지면
// 에이전트가 무엇을 고쳐야 하는지 모르고, 되먹임이 LLM 의 의견이 아니라
// 검증기의 출력이라는 것(OpenHands 와의 차이)이 뜻을 잃는다.
func TestTypeViolationNamesThePlaceTheWantAndTheGot(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","properties":{
	  "name":{"type":"string"},"count":{"type":"integer"},"ratio":{"type":"number"},
	  "ok":{"type":"boolean"},"tags":{"type":"array"},"meta":{"type":"object"},
	  "nothing":{"type":"null"}}}`)

	cases := []struct {
		field    string
		doc      string
		wantType string // 스키마가 요구한 것
		gotType  string // typeName 이 실제 값을 부르는 이름
	}{
		{"name", `{"name":3}`, "string", "integer"},
		{"name", `{"name":3.5}`, "string", "number"},
		{"name", `{"name":null}`, "string", "null"},
		{"name", `{"name":true}`, "string", "boolean"},
		{"name", `{"name":[]}`, "string", "array"},
		{"name", `{"name":{}}`, "string", "object"},
		{"count", `{"count":3.5}`, "integer", "number"},
		{"count", `{"count":"3"}`, "integer", "string"},
		{"ratio", `{"ratio":"x"}`, "number", "string"},
		{"ok", `{"ok":"true"}`, "boolean", "string"},
		{"tags", `{"tags":{}}`, "array", "object"},
		{"meta", `{"meta":[]}`, "object", "array"},
		{"nothing", `{"nothing":1}`, "null", "integer"},
	}
	for _, c := range cases {
		v := Validate(sch, []byte(c.doc))
		if len(v) != 1 {
			t.Fatalf("%s: got %d violations, want exactly 1: %+v", c.doc, len(v), v)
		}
		if v[0].Path != c.field {
			t.Errorf("%s: path=%q, want %q", c.doc, v[0].Path, c.field)
		}
		if v[0].Want != "expected type "+c.wantType {
			t.Errorf("%s: want=%q, want %q", c.doc, v[0].Want, "expected type "+c.wantType)
		}
		if v[0].Got != c.gotType {
			t.Errorf("%s: got=%q, want %q", c.doc, v[0].Got, c.gotType)
		}
	}
}

// 타입이 맞으면 통과한다 — 정직한 답을 떨어뜨리지 않는다 (ADR-020 의 기준).
func TestEveryTypeInTheVocabularyAcceptsItsOwnValue(t *testing.T) {
	cases := map[string]string{
		`{"type":"string"}`:  `"x"`,
		`{"type":"integer"}`: `7`,
		`{"type":"number"}`:  `7.5`,
		`{"type":"boolean"}`: `false`,
		`{"type":"array"}`:   `[1,2]`,
		`{"type":"object"}`:  `{"a":1}`,
		`{"type":"null"}`:    `null`,
	}
	for s, doc := range cases {
		if v := Validate(mustSchema(t, s), []byte(doc)); len(v) != 0 {
			t.Errorf("%s rejected its own value %s: %v", s, doc, v)
		}
	}
	// 정수는 number 이기도 하다 — JSON 에 정수 타입이 따로 없다.
	if v := Validate(mustSchema(t, `{"type":"number"}`), []byte(`7`)); len(v) != 0 {
		t.Errorf("an integer was rejected as a number: %v", v)
	}
}

// 모르는 타입 이름은 아무것도 제약하지 않는다 — 조용히 떨어뜨리지 않는다.
//
// ADR-020 의 기준은 「정직하면 항상 통과」다. 검증기가 모르는 낱말을
// 만났을 때 거절하면 그 기준을 어긴다. 거절할 낱말은 CheckBoundary 가
// 계약 검증 시점(400)에 이미 가려낸다.
func TestUnknownTypeNameConstrainsNothing(t *testing.T) {
	if v := Validate(mustSchema(t, `{"type":"date-time"}`), []byte(`"2026-08-29"`)); len(v) != 0 {
		t.Fatalf("an unknown type name rejected an honest answer: %v", v)
	}
	if v := Validate(mustSchema(t, `{"type":"date-time"}`), []byte(`3`)); len(v) != 0 {
		t.Fatalf("an unknown type name rejected an honest answer: %v", v)
	}
}

// 배열 원소는 몇 번째가 틀렸는지까지 말한다.
//
// 자리를 안 말하면 원소가 열 개일 때 되먹임이 쓸모없다.
func TestArrayItemsReportTheIndexThatFailed(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","properties":{
	  "tags":{"type":"array","items":{"type":"string"}}}}`)

	v := Validate(sch, []byte(`{"tags":["a",3,"c",true]}`))
	if len(v) != 2 {
		t.Fatalf("got %d violations, want 2 (index 1 and 3): %+v", len(v), v)
	}
	if v[0].Path != "tags[1]" {
		t.Errorf("path=%q, want %q", v[0].Path, "tags[1]")
	}
	if v[0].Got != "integer" {
		t.Errorf("got=%q, want %q", v[0].Got, "integer")
	}
	if v[1].Path != "tags[3]" {
		t.Errorf("path=%q, want %q", v[1].Path, "tags[3]")
	}
	if v[1].Got != "boolean" {
		t.Errorf("got=%q, want %q", v[1].Got, "boolean")
	}
	// 전부 맞으면 조용하다.
	if got := Validate(sch, []byte(`{"tags":["a","b"]}`)); len(got) != 0 {
		t.Errorf("a well-typed array was rejected: %v", got)
	}
}

// 중첩된 자리는 점으로 이어 붙인 경로를 낸다 — 루트의 이름만 주면 못 찾는다.
func TestNestedPathIsJoinedWithDots(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","properties":{
	  "outer":{"type":"object","required":["inner"],
	           "properties":{"deep":{"type":"object","properties":{"leaf":{"type":"string"}}}}}}}`)

	v := Validate(sch, []byte(`{"outer":{"deep":{"leaf":9}}}`))
	if len(v) != 2 {
		t.Fatalf("got %d violations, want 2 (missing inner, bad leaf): %+v", len(v), v)
	}
	paths := map[string]bool{}
	for _, x := range v {
		paths[x.Path] = true
	}
	for _, want := range []string{"outer.inner", "outer.deep.leaf"} {
		if !paths[want] {
			t.Errorf("path %q is missing from %+v", want, v)
		}
	}
}

// additionalProperties:false 는 모르는 키를 그 키의 이름으로 지적한다.
//
// 「어딘가에 남는 키가 있다」로는 못 고친다.
func TestExtraKeyIsNamedWhenAdditionalPropertiesIsFalse(t *testing.T) {
	sch := mustSchema(t, `{"type":"object","additionalProperties":false,
	  "properties":{"status":{"type":"string"}}}`)

	v := Validate(sch, []byte(`{"status":"found","confidence":0.8}`))
	if len(v) != 1 {
		t.Fatalf("got %d violations, want exactly 1: %+v", len(v), v)
	}
	if v[0].Path != "confidence" {
		t.Errorf("path=%q, want %q", v[0].Path, "confidence")
	}
	if !strings.Contains(v[0].Want, "not allowed") {
		t.Errorf("want=%q does not say the key is not allowed", v[0].Want)
	}
	// additionalProperties 를 안 적으면 여분의 키는 통과한다 — 기본이 열림이다.
	open := mustSchema(t, `{"type":"object","properties":{"status":{"type":"string"}}}`)
	if got := Validate(open, []byte(`{"status":"found","confidence":0.8}`)); len(got) != 0 {
		t.Errorf("an extra key was rejected without additionalProperties:false: %v", got)
	}
}

// 스키마 자리에 객체가 아닌 것이 오면 아무것도 제약하지 않는다 — 패닉이 아니다.
//
// JSON Schema 는 true/false 를 스키마로 허용한다. 계약이 그것을 실어 보내도
// 검증기가 죽으면 안 되고, 조용히 전부 거절해도 안 된다.
func TestNonObjectSchemaConstrainsNothingAndDoesNotPanic(t *testing.T) {
	if v := Validate(true, []byte(`{"anything":1}`)); len(v) != 0 {
		t.Fatalf("a boolean schema rejected a document: %v", v)
	}
	nested := mustSchema(t, `{"type":"object","properties":{"free":true}}`)
	if v := Validate(nested, []byte(`{"free":{"whatever":[1,2]}}`)); len(v) != 0 {
		t.Fatalf("a boolean sub-schema rejected a value: %v", v)
	}
}

// JSON 으로 못 그리는 값이 스키마에 섞여도 메시지가 빈 채로 나가지 않는다.
//
// 계약은 JSON 으로 들어오지만 Validate 는 any 를 받으므로 코드가 지은
// 스키마도 온다. 그때 fmtAny 가 죽거나 빈 문자열을 주면 위반 내역이
// 「무엇과 비교해 틀렸는가」를 잃는다.
func TestUnrenderableEnumStillProducesAReadableViolation(t *testing.T) {
	sch := map[string]any{"enum": []any{math.NaN()}}

	v := Validate(sch, []byte(`"found"`))
	if len(v) != 1 {
		t.Fatalf("got %d violations, want exactly 1: %+v", len(v), v)
	}
	if v[0].Want == "enum " || strings.TrimSpace(v[0].Want) == "enum" {
		t.Fatalf("the enum rendered to nothing, so the message says what it compared against: %q", v[0].Want)
	}
	if !strings.Contains(v[0].Want, "NaN") {
		t.Errorf("want=%q does not fall back to Go's own rendering of the value", v[0].Want)
	}
	if v[0].Got != `"found"` {
		t.Errorf("got=%q, want %q", v[0].Got, `"found"`)
	}
}

// typeName 은 JSON 밖의 값에도 이름을 준다 — 빈 문자열이나 패닉이 아니다.
//
// Validate 가 any 를 받으므로 이 갈래에 실제로 닿을 수 있고, Got 가 비면
// 위반 내역이 「무엇이 왔는가」를 잃는다.
func TestTypeNameHasAFallbackForValuesOutsideJSON(t *testing.T) {
	if got := typeName(struct{ A int }{1}); got != "?" {
		t.Fatalf("typeName(struct) = %q, want %q", got, "?")
	}
	if got := typeName(make(chan int)); got != "?" {
		t.Fatalf("typeName(chan) = %q, want %q", got, "?")
	}
}

// 허용 어휘 밖의 낱말은 「모르는 낱말」로 거절된다 — 조용히 무시하지 않는다.
//
// 거절 목록(minimum 등)에 없다고 통과시키면 새 판정 낱말이 목록 갱신
// 전까지 그대로 새어 들어온다. ADR-020 은 허용 목록 쪽을 좁혔다.
func TestUnknownKeywordIsRefusedNotIgnored(t *testing.T) {
	cases := map[string]string{
		`{"type":"object","$ref":"#/definitions/x"}`:                    "$ref",
		`{"type":"object","allOf":[{"type":"string"}]}`:                 "allOf",
		`{"type":"object","properties":{"a":{"const":"only-this"}}}`:    "const",
		`{"type":"array","items":{"type":"string","uniqueItems":true}}`: "uniqueItems",
	}
	for s, kw := range cases {
		err := CheckBoundary(mustSchema(t, s))
		if err == nil {
			t.Fatalf("the unknown keyword %s passed the boundary check", kw)
		}
		if !strings.Contains(err.Error(), kw) {
			t.Errorf("the offending keyword %s is not named: %v", kw, err)
		}
		if !strings.Contains(err.Error(), "allowed vocabulary") {
			t.Errorf("%s: the error does not say it is outside the allowed vocabulary: %v", kw, err)
		}
	}
}
