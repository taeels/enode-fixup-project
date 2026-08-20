package enode

import (
	"context"
	"os"
	"runtime"
	"sort"
	"strings"
)

// ★ R1 — 하네스에 넘길 환경변수는 화이트리스트다 ★
//
// 고치기 전 코드는 이랬다.
//
//	cmd.Env = append(os.Environ(), "OUT="+out, "IN="+in)
//	                 ^^^^^^^^^^^
//	★ ENODE_TOKEN 이 에이전트 손에 그대로 간다 ★
//
// 토큰을 쥔 에이전트는 자기를 노드로 광고하고 Run 을 claim 할 수 있다.
// 그러면 `leases(node_id) PRIMARY KEY` 로 지켜둔 I1 이 무의미해진다 —
// 열쇠를 쥔 쪽이 늘어나기 때문이다. 불변식을 코드가 아니라 인프라로 지킨 것의
// 값은 「열쇠가 하나」라는 전제 위에 있다.
//
// ★ 블랙리스트가 아니라 화이트리스트인 이유 ★
//
//	블랙리스트는 ★ 잊는다 ★. 나중에 ENODE_MEDIATOR 를 추가하면
//	막는 목록에 넣는 것을 잊어도 아무 일도 안 일어나 보인다 —
//	조용히 새는 것이 가장 나쁜 실패다.
//	화이트리스트는 반대로 틀리면 ★ 하네스가 안 돈다 ★. 닫히는 쪽으로 틀린다.

// baseAllow 는 하네스가 무엇이든 있어야 도는 것들이다.
//
// 하나씩 이유가 있다. 이유를 못 대는 이름은 여기 없다.
var baseAllow = []string{
	"PATH",     // 하네스 실행파일을 못 찾는다
	"HOME",     // ★ 자격증명이 여기 있다 ★ — MVP 의 transparent 인증이 곧 이것
	"USER",     // git 이 참조한다
	"LOGNAME",  //  〃
	"LANG",     // 인코딩. 없으면 한글 경로·로그가 깨진다
	"LC_ALL",   //  〃
	"LC_CTYPE", //  〃
	"TZ",       // 로그 시각
	"TERM",     // 없어도 돌지만 도구가 참조한다

	// ★ 사내망에서 이게 없으면 하네스가 밖으로 못 나간다 ★
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
	"http_proxy", "https_proxy", "no_proxy",
}

// windowsAllow — 윈도우는 이것들이 없으면 ★ 프로세스 생성 자체가 실패한다 ★.
// SYSTEMROOT 가 없으면 소켓조차 못 연다.
var windowsAllow = []string{
	"SYSTEMROOT", "SYSTEMDRIVE", "WINDIR", "COMSPEC", "PATHEXT",
	"TEMP", "TMP", "USERPROFILE", "APPDATA", "LOCALAPPDATA",
	"NUMBER_OF_PROCESSORS", "PROCESSOR_ARCHITECTURE",
}

// RunIdentity 는 인증 주입이 판단 재료로 쓸 것들이다 (R2).
//
// 지금은 아무도 안 본다. ★ 자리를 파는 이유 ★ 는, 나중에 「요청자 신원으로
// 돌린다」를 넣을 때 Credentials 구현만 갈아끼우면 되게 하기 위해서다.
// 이 구조체가 없으면 그때 runner 시그니처부터 고쳐야 한다.
type RunIdentity struct {
	RunID     string // 어느 Run 인가
	Step      string // 어느 단계인가
	Requester string // ★ runctl 로 요청한 사람 ★ (~/.gitconfig 이메일, ADR-015)
	NodeOwner string // ★ 이 노드를 띄운 사람 ★ — 요청자와 다를 수 있다
}

// Credentials 는 ★ 인증 주입 자리 ★ 다 (R1 · R2).
//
//	MVP    transparent — 머신에 이미 있는 자격증명을 그대로 쓴다.
//	       HOME 이 화이트리스트에 있다는 것이 곧 그 뜻이다.
//	나중    요청자 신원 / 팀 공용 신원 — 여기서 환경변수를 돌려주면 된다.
type Credentials interface {
	For(ctx context.Context, id RunIdentity) (map[string]string, error)
}

// Transparent 는 MVP 구현이다 — ★ 아무것도 주입하지 않는다 ★.
//
// nil 을 넘겨도 되게 하지 않고 굳이 타입을 둔 이유는, 「주입 안 함」이
// ★ 결정 ★ 이라는 것을 코드에 남기기 위해서다. nil 은 잊은 것과 구별되지 않는다.
type Transparent struct{}

func (Transparent) For(context.Context, RunIdentity) (map[string]string, error) {
	return nil, nil
}

// harnessEnv 는 하네스에 넘길 환경을 ★ 처음부터 조립한다 ★.
//
// os.Environ() 을 걸러내는 게 아니라 빈 것에서 쌓는다 — 걸러내기로 쓰면
// 새 변수가 생겼을 때 기본이 「통과」가 되고, 그건 위에 적은 조용한 누출이다.
//
//	extra   하네스가 선언한 이름 (Harness.Env()) — 예: ANTHROPIC_API_KEY
//	fixed   우리가 못 박는 것 — OUT · IN
//	inject  Credentials 가 돌려준 것. ★ 마지막에 얹혀 이긴다 ★
func harnessEnv(extra []string, fixed, inject map[string]string) []string {
	allow := map[string]bool{}
	for _, k := range baseAllow {
		allow[k] = true
	}
	if runtime.GOOS == "windows" {
		for _, k := range windowsAllow {
			allow[k] = true
		}
	}
	for _, k := range extra {
		if k = strings.TrimSpace(k); k != "" {
			allow[k] = true
		}
	}

	out := map[string]string{}
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		k := kv[:i]
		// 윈도우는 환경변수 이름이 대소문자를 안 가린다. 리눅스는 가린다.
		// 화이트리스트 조회를 OS 규칙에 맞춘다.
		if allow[k] || (runtime.GOOS == "windows" && allow[strings.ToUpper(k)]) {
			out[k] = kv[i+1:]
		}
	}
	for k, v := range fixed {
		out[k] = v
	}
	for k, v := range inject { // ★ 주입이 마지막이다 ★
		out[k] = v
	}

	kv := make([]string, 0, len(out))
	for k, v := range out {
		kv = append(kv, k+"="+v)
	}
	sort.Strings(kv) // 결정적 순서 — 기록을 비교할 수 있어야 한다
	return kv
}

// notablePrefix 는 ★ 인증 구성처럼 보이는 ★ 이름들이다.
//
// 부모 환경에 있었지만 화이트리스트에 없어 안 넘긴 것을 골라내 로그에 남긴다.
// 값은 안 남긴다 — ★ 이름만 ★ 이다.
//
// 이게 필요한 이유: 화이트리스트는 틀리면 「하네스가 안 돈다」로 나타나는데,
// 그때 사람이 보는 것은 "인증이 없다" 는 하네스의 메시지뿐이라
// ★ 우리가 걸렀다는 사실이 어디에도 안 보인다 ★. 그게 조용한 실패다.
var notablePrefix = []string{"ANTHROPIC_", "CLAUDE_", "AWS_", "GOOGLE_", "GEMINI_", "OPENAI_", "AZURE_"}

// droppedNotable 은 안 넘긴 것 중 인증처럼 보이는 이름을 돌려준다.
//
// ★ ENODE_ 는 일부러 뺀다 ★ — 그건 안 넘기는 게 의도이고, 매번 로그에 찍히면
// 진짜 신호가 묻힌다.
func droppedNotable() []string {
	allow := map[string]bool{}
	for _, k := range baseAllow {
		allow[k] = true
	}
	if runtime.GOOS == "windows" {
		for _, k := range windowsAllow {
			allow[k] = true
		}
	}
	// ★ 등록된 하네스 전부의 선언을 본다 ★ — 어댑터가 늘어도 여기를 안 고친다.
	for _, h := range harnesses {
		for _, k := range h.Env() {
			allow[k] = true
		}
	}

	var out []string
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		k := kv[:i]
		if allow[k] {
			continue
		}
		for _, p := range notablePrefix {
			if strings.HasPrefix(k, p) {
				out = append(out, k)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// commandEnv 는 ★ 명령 단계 ★ 가 흔히 필요로 하는 이름이다.
//
// 에이전트와 명령은 위협은 같지만 필요한 환경의 폭이 다르다 —
// 빌드는 툴체인 변수를 요구한다. 그래서 기본 목록이 따로 있고,
// ★ 부족한 것은 계약이 steps[].env 로 이름을 적어 더한다 ★.
//
// ★ 값이 아니라 이름인 이유 ★ — 값을 계약에 적으면 그 계약이 Run Record 의
// manifest 로 봉인되어(ADR-005) 자격증명이 영구히 남는다.
var commandEnv = []string{
	// 커널 크로스 빌드
	"ARCH", "CROSS_COMPILE", "KBUILD_OUTPUT", "KBUILD_BUILD_TIMESTAMP",
	"CC", "CXX", "LD", "AR", "NM", "OBJCOPY", "OBJDUMP", "STRIP",
	"CFLAGS", "LDFLAGS", "MAKEFLAGS", "JOBS",
	// 캐시 — ★ 없으면 증분 빌드가 느려진다 ★ (ADR-007 이 준비물로 잡은 것)
	"CCACHE_DIR", "SCCACHE_DIR",
	// 흔한 도구
	"SHELL", "PWD",
}
