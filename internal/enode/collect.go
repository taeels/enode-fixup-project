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
// 이유가 구조적이다 — 모델이 쓸 수 있는 곳과
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
// ★ 어휘적 비교로는 부족하다 ★ — filepath.Abs 는 심링크를 안 푼다. 그래서
// 최종 항목만 Lstat 으로 걸러내면 ★ 부모 디렉터리가 심링크인 경우가 통과한다 ★:
//
//	저장소 안에  x -> /etc  가 들어 있다        ★ git 은 심링크를 담는다 ★
//	collect: { "leak": "x/shadow" }
//	  ├ badPattern  ".." 이 없다                 통과
//	  ├ Lstat       x/shadow 는 ★ 일반 파일 ★    통과 (심링크는 부모인 x 다)
//	  └ Abs/Rel     "x/shadow"                   ★ 워크스페이스 안으로 판정 ★
//	⇒ ★ 워크스페이스 밖 파일이 Record 에 봉인된다 ★
//
// 계약은 ★ 노드 주인이 아닌 사람 ★ 이 내고, 워크스페이스는 ★ 리뷰 대상 코드 ★ 다.
// 둘 다 신뢰할 수 없으므로 ★ 실경로로 비교한다 ★.
//
// 실경로로 검사하고 ★ 원래 경로로 여는 것 ★ 은 여기서 안전하다 — collect 는
// 명령이 ★ 끝난 뒤 ★ 에 돌아서 심링크를 바꿔칠 프로세스가 남아 있지 않다.
// (그래서 검사-후-사용 경쟁이 성립하지 않는다.)
func onlyRegularInside(ws string, hits []string) []string {
	rootReal, err := realPath(ws)
	if err != nil {
		return nil
	}
	var keep []string
	for _, h := range hits {
		fi, err := os.Lstat(h)
		if err != nil || !fi.Mode().IsRegular() { // 심링크·디렉터리·장치 제외
			continue
		}
		resolved, err := realPath(h)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(rootReal, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			continue
		}
		keep = append(keep, h)
	}
	sort.Strings(keep)
	return keep
}

// realPath 는 절대경로로 만든 뒤 ★ 심링크를 푼다 ★.
//
// ★ 양쪽을 다 풀어야 한다 ★ — 워크스페이스 자신이 심링크 아래 있을 수 있어서
// (/tmp → /private/tmp 같은 환경) 한쪽만 풀면 정상 산출물까지 떨어뜨린다.
func realPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
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

// sealInput 은 ★ $IN 을 읽기 전용으로 잠근다 ★ (2026-08-20).
//
// ★ 2026-08-22 — 이 함수의 무게가 늘었다 ★ (ADR-042)
//
// 아래 논거는 --add-dir 이 그은 경계를 전제로 쓰였는데, 권한 모드가
// bypassPermissions 로 바뀌면서 ★ 그 경계가 없어졌다 ★. 그래도 이 잠금은
// 살아 있다 — ★ 권한 모드가 아니라 파일시스템이 거는 것 ★ 이기 때문이다.
// 달라진 것은 ★ 이제 이것이 $IN 에 대한 유일한 기계적 방어 ★ 라는 점이다.
//
// ★ 왜 필요한가 — R7 의 포함관계가 $IN 만큼 깨져 있었다 ★
//
//	모델이 쓸 수 있는 곳  { 워크스페이스, $OUT, ★$IN★ }   claude.go 의 --add-dir
//	훅이 볼 수 있는 곳    { 워크스페이스, $OUT       }   hook.go 의 HookArgs
//	                                        ▲
//	                                        └── 여기서 포함관계가 깨진다
//
// --add-dir 은 ★ 읽기와 쓰기를 안 가른다 ★. 그런데 $IN 은 읽기만 필요하다 —
// 이전 단계 산출물이 깔리는 곳이고, 시연에 대입하면 ④의 리뷰 대상 diff 와
// ⑥의 되먹인 빌드 오류 로그다. ★ 에이전트가 자기가 반증할 증거를 고쳐 쓸 수 있다 ★.
//
// ★ 관측을 늘리는 대신 집합을 좁힌다 ★ — 훅이 $IN 도 보게 하면 "변했다" 를
// 알아챌 뿐이지만, 잠그면 애초에 안 변한다. Agent SDK 보안 문서가 읽기 전용
// 마운트(-v …:ro)로 같은 답을 낸다. 우리는 마운트가 아니라 임시 디렉터리라 권한으로 한다.
//
//	파일     0444   Write · Edit 가 실패한다
//	디렉터리 0555   ★ 새 파일 생성과 unlink 를 막는다 ★ — 파일만 잠그면
//	                지우고 다시 만들 수 있어 잠금이 무의미해진다
//
// ★ 사정거리를 과장하지 않는다 ★ — 우리와 같은 uid 로 도는 Bash 는 chmod 로
// 되돌릴 수 있다. 이 잠금이 확실히 덮는 것은 ★ 하네스의 파일 도구 ★ 다.
// ADR-042 이후로는 Bash 가 열려 있으므로 ★ 이것은 방어가 아니라 표지 ★ 에
// 가깝다 — 우연한 덮어쓰기는 막고 의도적인 것은 못 막는다. 그것이 곧
// 「노드는 소유자가 신뢰하는 계약만 받는다」가 지는 몫이다.
func sealInput(dir string) string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err.Error()
	}
	for _, e := range ents {
		if e.IsDir() {
			continue // MVP 의 $IN 은 평평하다 — 트리가 생기면 그때 재귀한다
		}
		if err := os.Chmod(filepath.Join(dir, e.Name()), 0o444); err != nil {
			return err.Error()
		}
	}
	// ★ 디렉터리는 마지막에 ★ — 먼저 잠그면 위의 Chmod 가 막힌다.
	if err := os.Chmod(dir, 0o555); err != nil {
		return err.Error()
	}
	return ""
}
