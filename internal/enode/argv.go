package enode

import "strings"

// argv 의 $OUT · $IN 을 푼다
//
// `run` 은 argv 배열이라 셸을 안 거친다 (리다이렉션·글롭이 없고 문자열 조립으로
// 인젝션이 생길 자리도 없다). 그런데 그 대가로 `$OUT` 이 리터럴로 넘어간다.
//
// 이게 왜 중요한가 — 「누가 산출물 경로를 아는가」
//
//	우리 규약은 "$OUT 에 평평한 이름으로 낸다" 이고, 누군가는 빌드 산출물의
//	위치를 그 이름에 매핑해야 한다:
//
//	  스크립트가 한다       cp arch/arm/boot/zImage $OUT/artifact
//	  계약이 한다           steps[].collect            ← 경로를 미리 알아야 한다
//	  빌드가 직접 놓는다 make modules_install INSTALL_MOD_PATH=$OUT
//
// 세 번째가 제일 낫다 — 아무도 경로를 미리 몰라도 된다. 빌드 시스템이
// 이미 "여기 놓아라" 를 지원하기 때문이다(INSTALL_MOD_PATH · DESTDIR · -o …).
// 그런데 그러려면 argv 에서 $OUT 이 풀려야 한다. 그래서 이 함수가 있다.
//
// 셸을 들이지 않는다 — 두 이름만 안다. 그 외의 $VAR 는 손대지 않는다:
// 셸 확장을 흉내 내기 시작하면 인용·글롭·워드 분할이 줄줄이 따라오고,
// argv 배열을 고른 이유가 사라진다.
func expandIO(argv []string, io IOPaths) []string {
	if len(argv) == 0 {
		return argv
	}
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = expandOne(a, io)
	}
	return out
}

func expandOne(s string, io IOPaths) string {
	for _, r := range []struct{ from, to string }{
		{"${OUT}", io.Out}, {"$OUT", io.Out},
		{"${IN}", io.In}, {"$IN", io.In},
	} {
		if r.to != "" {
			s = strings.ReplaceAll(s, r.from, r.to)
		}
	}
	return s
}

// unexpandedVars 는 안 풀린 $VAR 가 남았는지 본다.
//
// 계약 저자가 `$WORKSPACE` 나 `$HOME` 을 쓰면 셸이 없으므로 리터럴로 간다.
// 조용히 틀리게 두지 않는다 — 이름을 로그에 남겨 사람이 알아채게 한다.
// 막지는 않는다: 진짜 `$` 를 인자로 넘기는 경우가 있을 수 있고,
// 여기서 단계를 실패시키면 우리가 판정하는 것이 된다 (ADR-004 · I3).
func unexpandedVars(argv []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range argv {
		for i := 0; i < len(a); i++ {
			if a[i] != '$' {
				continue
			}
			j := i + 1
			if j < len(a) && a[j] == '{' {
				j++
			}
			k := j
			for k < len(a) && (a[k] == '_' ||
				(a[k] >= 'A' && a[k] <= 'Z') || (a[k] >= 'a' && a[k] <= 'z') ||
				(a[k] >= '0' && a[k] <= '9')) {
				k++
			}
			if k > j {
				name := a[j:k]
				if !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
			i = k - 1
		}
	}
	return out
}
