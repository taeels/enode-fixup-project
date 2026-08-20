package enode

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ★ steps[].collect — 셋 중 마지막 수단이다 ★
//
// 우리 규약이 "$OUT 에 평평한 이름으로 낸다" 이므로, 누군가는 빌드 산출물의
// ★ 위치 ★ 를 ★ 우리 이름 ★ 에 매핑해야 한다. 방법이 셋이고 ★ 순서가 있다 ★:
//
//	① ★ 빌드가 직접 놓는다 ★   make modules_install INSTALL_MOD_PATH=$OUT
//	                            gcc -o $OUT/prog · DESTDIR=$OUT · --output-dir …
//	   ★ 아무도 경로를 미리 몰라도 된다 ★ — 빌드 시스템이 이미 지원한다.
//	   argv.go 의 $OUT 치환이 이걸 가능하게 한다. ★ 이게 기본이다. ★
//
//	② 스크립트가 옮긴다        cp arch/arm/boot/zImage $OUT/artifact
//	   빌드가 놓을 곳을 못 받을 때. ★ 빌드 지식 옆에 있어 제일 정확하다. ★
//
//	③ collect                  계약이 경로를 적는다
//	   ★ 스크립트를 못 고칠 때만 ★ — 사내 빌드 스크립트가 남의 것이고
//	   $OUT 규약을 모르는 경우.
//
// ★ ③이 마지막인 이유 ★ — 경로를 ★ 미리 알아야 한다 ★. 커널 버전 · CONFIG ·
// O= 빌드에 따라 변하는 것을 계약에 못 박는 것은 작위적이다. 그래서 틀려도
// 안전하게 틀리도록 만들었다: 못 찾으면 안 걷고 이유만 적으며(새 실패 경로 없음),
// 스크립트가 낸 것이 우선이라 ①②와 같이 써도 안 깨진다.
//
// 남는 값 하나는 ★ 출처가 기록에 남는다 ★ 는 것이다 — cp 는 스크립트 안에 숨어
// Record 만 봐선 blob 이 어디서 왔는지 모른다 (ADR-005 성질 4).

// collectNote 는 왜 못 걷었는지다. ★ 판정이 아니라 사실이다 ★.
type collectNote struct{ Name, Why string }

// collectDeclared 는 계약이 적은 경로를 $OUT 으로 옮긴다.
//
// ★ 새 실패 경로를 만들지 않는다 ★ — 못 걷으면 produced 가 불만족이 되고
// 판정은 success_when 이 한다 (ADR-004 · I3). 여기서는 이유만 남긴다.
func collectDeclared(ws, out string, spec map[string]string) (got []string, notes []collectNote) {
	if ws == "" || len(spec) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(spec))
	for n := range spec {
		names = append(names, n)
	}
	sort.Strings(names) // 결정적 순서 — 기록을 비교할 수 있어야 한다

	for _, name := range names {
		pat := spec[name]
		if why := badPattern(pat); why != "" {
			notes = append(notes, collectNote{name, why})
			continue
		}
		// ★ 이미 있으면 안 덮어쓴다 ★ — 스크립트가 직접 낸 것이 우선. collect 는 보조다.
		dst := filepath.Join(out, name)
		if _, err := os.Lstat(dst); err == nil {
			continue
		}

		hits, err := filepath.Glob(filepath.Join(ws, pat))
		if err != nil {
			notes = append(notes, collectNote{name, "글롭이 이상하다: " + err.Error()})
			continue
		}
		hits = onlyRegularInside(ws, hits)
		switch {
		case len(hits) == 0:
			notes = append(notes, collectNote{name,
				fmt.Sprintf("%q 에 맞는 파일이 없다", pat)})
		case len(hits) > 1:
			// ★ 하나의 blob 이름에 여럿을 넣지 않는다 ★ — 소비자가 예측을 못 한다
			// (어떨 땐 .ko, 어떨 땐 묶음). 계약 저자가 글롭을 좁히는 것이 맞다.
			rel := make([]string, len(hits))
			for i, h := range hits {
				rel[i], _ = filepath.Rel(ws, h)
			}
			if len(rel) > 6 {
				rel = append(rel[:6], fmt.Sprintf("…외 %d개", len(hits)-6))
			}
			notes = append(notes, collectNote{name,
				fmt.Sprintf("%q 에 %d개가 맞는다 (%s) — 하나만 맞게 좁혀라",
					pat, len(hits), strings.Join(rel, ", "))})
		default:
			if err := copyFile(hits[0], dst); err != nil {
				notes = append(notes, collectNote{name, "옮기지 못했다: " + err.Error()})
				continue
			}
			got = append(got, name)
		}
	}
	return got, notes
}

// badPattern 은 ★ 워크스페이스 밖을 가리키는 것을 막는다 ★.
//
// 이유가 구조적이다 — acceptEdits + --add-dir 로 모델이 쓸 수 있는 곳을 좁혀놨는데,
// collect 가 아무 데나 가리키면 ★ 계약이 그 경계를 우회한다 ★.
// 그리고 계약은 ★ 노드 주인이 아닌 사람 ★ 이 낸다.
func badPattern(pat string) string {
	if pat == "" {
		return "경로가 비었다"
	}
	if filepath.IsAbs(pat) || strings.HasPrefix(pat, "/") || strings.HasPrefix(pat, `\`) {
		return "절대경로는 안 된다 — 워크스페이스 상대경로만 쓴다"
	}
	if vol := filepath.VolumeName(pat); vol != "" {
		return "드라이브 지정은 안 된다 — 워크스페이스 상대경로만 쓴다"
	}
	clean := filepath.ToSlash(filepath.Clean(pat))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "워크스페이스 밖을 가리킬 수 없다"
	}
	return ""
}

// onlyRegularInside 는 ★ 일반 파일 ★ 이면서 ★ 워크스페이스 안 ★ 인 것만 남긴다.
//
// 심링크를 따라가면 밖으로 나갈 수 있으므로 Lstat 으로 걸러낸다 —
// 글롭이 워크스페이스 안이어도 그 심링크가 가리키는 곳은 밖일 수 있다.
func onlyRegularInside(ws string, hits []string) []string {
	rootAbs, err := filepath.Abs(ws)
	if err != nil {
		return nil
	}
	var keep []string
	for _, h := range hits {
		fi, err := os.Lstat(h)
		if err != nil || !fi.Mode().IsRegular() { // 심링크·디렉터리·장치 제외
			continue
		}
		abs, err := filepath.Abs(h)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(rootAbs, abs)
		if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			continue
		}
		keep = append(keep, h)
	}
	sort.Strings(keep)
	return keep
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck
	tmp := dst + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, in); err != nil {
		f.Close()      //nolint:errcheck
		os.Remove(tmp) //nolint:errcheck
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return err
	}
	// ★ 이름 바꾸기로 마무리한다 ★ — 반쯤 쓴 파일이 산출물로 걷히면 안 된다.
	return os.Rename(tmp, dst)
}
