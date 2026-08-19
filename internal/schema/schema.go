// Package schema 는 산출물의 ★ 형식만 ★ 검증한다 (ADR-020).
//
// ★ 왜 JSON Schema 전체를 쓰지 않는가 ★
//
// ADR-020 이 경계선을 그었다:
//
//	기준: 에이전트가 정직하게 답했을 때 통과하지 못할 수 있으면 그건 ★ 판정 ★ 이다
//	  ○ enum            정직하면 항상 통과       — 형식
//	  ✗ minimum         낮게 답하면 실패         — 판정
//	  ✗ minLength       길이로 품질을 잰다       — 판정
//
// 그 경계를 ★ 산문으로 두면 새어나간다 ★. 전체 JSON Schema 를 받으면
// 누군가 confidence >= 0.8 을 쓰고, 그 순간 ADR-004(기계적 판정만)가
// 스키마를 통해 무너진다. 그래서 허용 어휘를 좁히고 ★ 나머지는 400 으로 거절한다 ★.
// I1 을 기본키로, I4 를 chmod 로 강제한 것과 같은 결이다.
package schema

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// 허용하는 것 — 형식만 말한다.
var allowed = map[string]bool{
	"type": true, "required": true, "properties": true, "enum": true,
	"items": true, "additionalProperties": true,
	"title": true, "description": true, // 사람이 읽는 것. 검증에 안 쓴다.
}

// 거절하는 것 — 정직한 답을 떨어뜨릴 수 있다.
var rejected = map[string]string{
	"minimum": "값의 크기로 판정한다", "maximum": "값의 크기로 판정한다",
	"exclusiveMinimum": "값의 크기로 판정한다", "exclusiveMaximum": "값의 크기로 판정한다",
	"multipleOf": "값으로 판정한다",
	"minLength":  "길이로 품질을 잰다", "maxLength": "길이로 품질을 잰다",
	"minItems": "개수로 판정한다", "maxItems": "개수로 판정한다",
	"pattern":       "내용으로 판정한다 — 어휘를 좁히려면 enum 을 쓴다",
	"format":        "내용으로 판정한다",
	"minProperties": "개수로 판정한다", "maxProperties": "개수로 판정한다",
}

// CheckBoundary 는 스키마가 ★ 형식만 제약하는지 ★ 본다.
// 계약 검증 시점(400)에 부른다 — 실행하고 나서 알면 늦다.
func CheckBoundary(s any) error { return boundary(s, "") }

func boundary(s any, path string) error {
	m, ok := s.(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 에러 메시지를 결정적으로
	for _, k := range keys {
		if why, bad := rejected[k]; bad {
			return fmt.Errorf("스키마 %s%s 는 쓸 수 없다 — %s (ADR-020: 스키마는 형식만 제약한다)",
				at(path), k, why)
		}
		if !allowed[k] {
			return fmt.Errorf("스키마 %s%s 는 허용 어휘가 아니다 (ADR-020)", at(path), k)
		}
	}
	if p, ok := m["properties"].(map[string]any); ok {
		names := make([]string, 0, len(p))
		for n := range p {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if err := boundary(p[n], path+"."+n); err != nil {
				return err
			}
		}
	}
	if it, ok := m["items"]; ok {
		return boundary(it, path+"[]")
	}
	return nil
}

func at(path string) string {
	if path == "" {
		return ""
	}
	return strings.TrimPrefix(path, ".") + " 의 "
}

// Violation 은 검증 실패 하나다. ★ feedback 으로 되먹여진다 ★ (ADR-013 의 루프) —
// 되먹이는 것이 LLM 의 의견이 아니라 ★ 검증기의 출력 ★ 이라는 것이 OpenHands 와의 차이다.
type Violation struct {
	Path string `json:"path"`
	Want string `json:"want"`
	Got  string `json:"got"`
}

func (v Violation) String() string {
	p := v.Path
	if p == "" {
		p = "(최상위)"
	}
	return fmt.Sprintf("%s: %s — 받은 것: %s", p, v.Want, v.Got)
}

// Validate 는 문서가 스키마를 만족하는지 본다.
func Validate(sch any, doc []byte) []Violation {
	var v any
	if err := json.Unmarshal(doc, &v); err != nil {
		return []Violation{{Path: "", Want: "JSON 이어야 한다", Got: err.Error()}}
	}
	var out []Violation
	check(sch, v, "", &out)
	return out
}

func check(sch, v any, path string, out *[]Violation) {
	m, ok := sch.(map[string]any)
	if !ok {
		return
	}
	if t, ok := m["type"].(string); ok && !typeOK(t, v) {
		*out = append(*out, Violation{path, "타입이 " + t, typeName(v)})
		return // 타입이 틀리면 아래를 볼 의미가 없다
	}
	if e, ok := m["enum"].([]any); ok {
		hit := false
		for _, want := range e {
			if equal(want, v) {
				hit = true
				break
			}
		}
		if !hit {
			*out = append(*out, Violation{path, "enum " + fmtAny(e), fmtAny(v)})
		}
	}
	obj, isObj := v.(map[string]any)
	if !isObj {
		if arr, isArr := v.([]any); isArr {
			if it, ok := m["items"]; ok {
				for i, e := range arr {
					check(it, e, fmt.Sprintf("%s[%d]", path, i), out)
				}
			}
		}
		return
	}
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			name, _ := r.(string)
			if _, has := obj[name]; !has {
				*out = append(*out, Violation{join(path, name), "필수다", "없음"})
			}
		}
	}
	props, _ := m["properties"].(map[string]any)
	if props != nil {
		names := make([]string, 0, len(obj))
		for n := range obj {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if p, ok := props[n]; ok {
				check(p, obj[n], join(path, n), out)
			} else if extra, ok := m["additionalProperties"].(bool); ok && !extra {
				*out = append(*out, Violation{join(path, n), "이 키는 허용되지 않는다", "있음"})
			}
		}
	}
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func typeOK(t string, v any) bool {
	switch t {
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "number":
		_, ok := v.(float64)
		return ok
	case "integer":
		f, ok := v.(float64)
		return ok && f == float64(int64(f))
	case "null":
		return v == nil
	}
	return true
}

func typeName(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		if x == float64(int64(x)) {
			return "integer"
		}
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "?"
}

func equal(a, b any) bool { return fmtAny(a) == fmtAny(b) }

func fmtAny(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
