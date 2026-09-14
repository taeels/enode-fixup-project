package enode

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// 하네스 구성요소 — 이 단계가 무엇을 열지는 exec 앞에서 정해진다
//
// 파일이 아니라 메모리다. resolveComponents 가 실패해도 아무것도 안 남고,
// 쓰는 쪽(Instrument)은 정책을 다시 판단하지 않는다 — 이미 정해진 것을 쓴다.
//
//	resolveComponents   정한다.  파일을 하나도 안 만진다
//	Instrument          쓴다.    무엇을 열지 안 정한다
//
// U4 가 출처 둘(노드 선언 · 워크스페이스)을 합쳤고 U5 가 팩을 더했다.
//
//	readWorkspaceMCP    읽는다.  가장자리다 — 정하지 않는다
//	readPack            읽는다.  순수하다 — 파일을 하나도 안 만진다.
//	                    여는 것은 runner.go 의 openPack 이다

// MCPServer 는 노드가 선언한 MCP 서버 하나다 (ADR-035 §4.4).
//
// stdio 와 remote 를 한 구조체가 받는다 — 하네스가 읽는 형식과 같아서
// 허용목록으로 옮겨 적는 것이 복사가 된다.
type MCPServer struct {
	Command    string   `yaml:"command,omitempty"`    // stdio
	Args       []string `yaml:"args,omitempty"`       // stdio
	URL        string   `yaml:"url,omitempty"`        // remote
	Credential string   `yaml:"credential,omitempty"` // remote. 환경변수 이름이다. 값이 아니다

	// Env 는 이름에서 이름으로 간다. 값이 아니다.
	//
	// 키는 하네스가 서버에게 줄 변수의 이름이고, 값은 노드 환경에서 그 값을
	// 길어 올 변수의 이름이다. 허용목록에는 ${이름} 참조로 나가고 실제 값은
	// R1 화이트리스트가 노드 환경으로 넘긴 것이다 — 그래서 값이 파일에 안 남는다.
	//
	// 검증은 config.go 의 validateEnvNames 가 한다. 여기서 안 하는 이유는
	// 이 형식이 노드 선언과 팩(U5) 둘 다에 쓰이는데, 팩의 것은 다른 시점에
	// 다른 문구로 거절되기 때문이다.
	Env map[string]string `yaml:"env,omitempty"`

	// Extra 는 우리가 모르는 키다 — 그대로 허용목록에 옮긴다 (decisions.md 2절).
	//
	// 왜 필드가 필요한가 — yaml 은 모르는 키를 말없이 버린다. 담을 자리가
	// 없으면 그 결정이 코드로 안 선다. 하네스가 키를 늘릴 때(transport ·
	// headers 같은) 우리 판을 갈아끼우지 않고 지나가는 길이 이 필드다.
	//
	// yaml:"-" 인 것은 읽기를 UnmarshalYAML 이 직접 하기 때문이다. 쓰기는
	// 안 짓는다 — setup 이 설정을 새로 지어 쓰기만 하고 읽어서 다시 쓰는
	// 경로가 저장소에 0 이라 오늘 잃을 것이 없다. 그 0 이 깨지는 날
	// MarshalYAML 이 함께 서야 한다.
	Extra map[string]any `yaml:"-"`
}

// knownMCPKeys 는 UnmarshalYAML 이 Extra 에서 덜어낼 이름이다.
//
// 구조체의 태그와 이 목록이 갈리면 아는 키가 Extra 로도 들어가 허용목록에
// 두 번 적힌다. 시험이 둘을 대조한다.
var knownMCPKeys = []string{"command", "args", "url", "credential", "env"}

// UnmarshalYAML 은 같은 노드를 두 번 푼다 (U3 · decisions.md 2절).
//
// 한 번만 풀면 둘 중 하나를 잃는다. 구조체로만 풀면 모르는 키가 사라지고,
// map 으로만 풀면 yaml 의 종류 검사가 사라져 args: "--x" 가 오류 대신
// 문자열로 들어온다. 그 검사는 우리가 다시 짤 것이 아니라 지킬 것이다.
func (s *MCPServer) UnmarshalYAML(value *yaml.Node) error {
	// shadow 는 이 메서드를 안 갖는 같은 모양이다 — 안 그러면 Decode 가
	// 자기 자신을 다시 불러 무한히 돈다.
	type shadow MCPServer
	var known shadow
	if err := value.Decode(&known); err != nil {
		return err
	}
	var all map[string]any
	if err := value.Decode(&all); err != nil {
		return err
	}
	for _, k := range knownMCPKeys {
		delete(all, k)
	}
	*s = MCPServer(known)
	if len(all) > 0 {
		s.Extra = all
	}
	return nil
}

// kind 는 이 선언이 어느 종류인가다. 로그가 읽는다.
//
// command 로 가른다 — validateMCP 가 둘 다 적힌 선언을 이미 거절하므로
// 이 갈래는 모호를 안 만난다.
func (s MCPServer) kind() string {
	if s.Command != "" {
		return "stdio"
	}
	return "remote"
}

// mcpUp 은 이 서버가 지금 뜨겠는가다 (FR-3 · ADR-035 §4.4).
//
// 실제 연결은 안 한다 — 광고마다 남의 서버를 두드리면 그 서버가 죽을 때
// 함대가 함께 죽는다 (requirements.md 4.4 의 순연).
//
// 「뜨나」는 존재이지 동작이 아니다. 셋을 못 잡는다 (decisions.md 6절 ⑭):
//
//	못 잡는다   자격증명이 틀렸다          환경변수 이름만 본다. 값을 안 본다
//	못 잡는다   엔드포인트가 죽었다        접근을 안 한다
//	못 잡는다   실행은 되나 MCP 가 아니다   CA2 의 가짜 서버 true 가 그 경우다
//
// 돌려주는 오류가 곧 노드 로그의 사유다 — 빠진 이유를 사람이 읽는다.
func mcpUp(s MCPServer) error {
	if s.Command != "" {
		if _, err := exec.LookPath(s.Command); err != nil {
			if strings.ContainsRune(s.Command, filepath.Separator) {
				return fmt.Errorf("command %q is not an executable file", s.Command)
			}
			return fmt.Errorf("command %q not found in PATH", s.Command)
		}
		return nil
	}
	// 인증 없는 endpoint 가 있다. 이름을 안 적었으면 볼 것이 없고, 볼 것이
	// 없는 것은 "안 뜬다" 가 아니다.
	if s.Credential == "" {
		return nil
	}
	if _, ok := os.LookupEnv(s.Credential); !ok {
		return fmt.Errorf("credential %s is not set in the node environment", s.Credential)
	}
	return nil
}

// mcpFP 는 뜨는 서버만 광고 속성으로 낸다 (FR-3).
//
// Fingerprinter 의 한 종류다. 프로세스를 안 띄우지만 호출 수가 노드 설정에
// 비례하므로 값싼 쪽에 안 둔다 (requirements.md 4.4 · components.md 2.2).
type mcpFP struct{}

func (mcpFP) Kind() string { return "mcp" }

func (mcpFP) Probe(_ context.Context, l Local, log *slog.Logger) (map[string]string, error) {
	if len(l.MCP) == 0 {
		// 선언이 없는 것은 정상이다. 로그도 안 낸다.
		return nil, nil
	}
	// 이름 순으로 돈다 — 같은 설정이면 같은 로그 순서다.
	names := make([]string, 0, len(l.MCP))
	for name := range l.MCP {
		names = append(names, name)
	}
	sort.Strings(names)

	attrs := map[string]string{}
	for _, name := range names {
		s := l.MCP[name]
		if err := mcpUp(s); err != nil {
			// 안 뜨는 서버는 이 종류의 실패가 아니라 정상적인 결과다 —
			// 빼고 보내는 것이 "지금은 못 한다" 다 (ADR-017 결정 3).
			// 그래도 조용히 빼지 않는다: 사람이 고칠 수 있는 문제다.
			log.Warn("mcp server is not up; dropping it from the advertisement",
				"server", name, "kind", s.kind(), "err", err)
			continue
		}
		// 값은 언제나 "1" 이다 — 매처가 완전 일치만 보므로 버전이나 경로를
		// 실으면 계약이 그 글자를 맞춰 적어야 한다.
		attrs["mcp."+name] = "1"
	}
	return attrs, nil
}

// Components 는 이 단계가 하네스에 실어 줄 것 전부다.
//
// Servers 는 MCPServer 가 아니라 허용목록 항목이다 (U4).
//
// 출처마다 어휘가 다르기 때문이다 — 노드 선언은 우리가 정의한 MCPServer 이고,
// 워크스페이스 .mcp.json 과 팩의 mcp.json 은 하네스가 정의한 형식이다. 그 둘을
// MCPServer 로 받으면 우리가 모르는 키(type · headers)가 사라지고 env 의 뜻이
// 뒤집힌다 — 노드 선언의 env 는 이름에서 이름으로 가는데 그 파일들의 env 는 값이다.
//
// 최종 항목으로 올리면 합치는 자리에서 어휘가 하나가 된다. 노드 것은
// allowlistEntry() 를 지나 들어오고 그 밖의 출처는 원문 그대로 들어온다 —
// 번역하는 코드가 없으므로 번역이 못 틀린다.
type Components struct {
	Servers map[string]map[string]any // 이름 -> 허용목록 항목
	Pack    *Pack                     // nil 이면 팩 없음
	Notes   []string                  // 로그로 낼 사실 — 이름만 담는다
}

// Pack 은 검증을 통과한 팩이다. tar 를 다시 안 연다.
//
// 파일이 아니라 메모리다 — 거부가 파일을 남기기 전에 일어나야 하기 때문이다
// (SEC-A). 쓰는 쪽(Instrument)은 검증을 다시 안 한다.
type Pack struct {
	// SHA256 은 $IN 에서 받은 바이트의 것이다. 압축된 원본 그대로다.
	//
	// 푼 뒤가 아닌 이유는 대조다 — 기록을 읽는 사람이 같은 Run 의 blob 을
	// 받아 sha256sum 으로 이 값과 맞출 수 있어야 한다. 푼 바이트로 재면
	// 그 대조가 안 선다.
	SHA256 string

	// Files 는 규약 안의 항목이다. 이름 순으로 담는다.
	Files []PackFile

	// MCP 는 팩 mcp.json 의 mcpServers 를 원문 그대로 담는다.
	//
	// MCPServer 가 아닌 이유는 워크스페이스 .mcp.json 과 같다 — 이 파일은
	// 하네스가 정의한 형식이고 우리 어휘가 아니다. 번역하는 코드가 없으므로
	// 번역이 못 틀린다.
	MCP map[string]map[string]any
}

// PackFile 은 펴질 파일 하나다.
//
// Mode 를 안 싣는다 — 규약 안의 항목이 전부 글자라(SKILL.md · <이름>.md)
// 실행 비트가 할 일이 0 이고, tar 의 모드를 그대로 쓰면 팩이 0777 파일을
// 계장 안에 남긴다. 받을 이유가 없는 값을 받는 것이다. 권한은 쓰는 쪽이
// 정한다 — 파일 0600 · 디렉터리 0700 으로 계장 안의 다른 파일과 같다.
type PackFile struct {
	Name string // 팩 안의 상대경로. skills/ 또는 agents/ 로 시작한다
	Data []byte
}

// PackLimits 는 푸는 쪽이 디스크와 메모리를 채우는 길을 막는다 (SEC-A).
type PackLimits struct {
	// MaxBytes 는 받은 바이트에도 푼 바이트에도 같이 건다.
	//
	// 둘 중 먼저 닿는 쪽에서 끊는다 — 받은 바이트만 재면 gzip 폭탄이
	// 통과하고, 푼 바이트만 재면 tar 꼬리에 붙은 거대한 쓰레기를 끝까지 읽는다.
	// 값이 하나라 「무엇을 넘었나」를 사람이 안 헷갈린다.
	MaxBytes int64
	// MaxFiles 는 항목 수다. 규약 밖 항목과 디렉터리까지 함께 센다.
	MaxFiles int
}

// defaultPackLimits 는 64 MiB 와 512 다.
//
// 전송은 이미 막혀 있다 — Mediator 의 MaxBlobBytes 기본 10 MiB 가 tar 자체를
// 막는다. 그 여섯 배를 푼 바이트에 준다. gzip 을 받으므로 압축률 6.4 까지가
// 그 안이고, 글자 팩의 실제 압축률이 거기 못 미친다 — 폭탄만 걸리고 성한
// 팩은 안 걸린다.
//
// 512 는 사고를 잡는 값이지 사람을 막는 값이 아니다 — 스킬 팩의 실제 파일
// 수는 열 단위다. 파일 하나의 상한은 안 둔다. 합계가 이미 그것을 진다.
var defaultPackLimits = PackLimits{MaxBytes: 64 << 20, MaxFiles: 512}

// packInput 은 가장자리가 연 결과다.
//
// runHarness 가 채우고 resolveComponents 가 등급과 문구를 정한다. 필드로
// Job 에 안 넣는다 — Job 은 claim.go 가 채우는 것이고 팩은 runner.go 가 연다.
// 섞으면 「claim.go 가 안 채웠다」는 사실이 안 보인다.
type packInput struct {
	Pack  *Pack
	Notes []string // 규약 밖 항목의 이름 등. readPack 이 낸다
	Err   error    // 못 열었거나 검증에 걸렸다
}

// resolveComponents 는 이 단계가 무엇을 열지 정한다 (FR-2 · FR-4 · FR-6).
//
// 파일을 하나도 안 만진다. 출처는 Job 과 packInput 이 들고 온다 — 여는 것은
// 가장자리(claim.go 와 runner.go 의 openPack)이고 고르는 것은 여기다. 그래서
// 시험이 하네스도 디스크도 안 쓰고, 오류가 곧 단계 실패다 — 부르는 쪽이
// exec 전에 부른다.
//
// 팩이 Job 의 필드가 아니라 둘째 인자인 이유는 출처의 주인이 다르기 때문이다 —
// Job 은 claim.go 가 채우고 팩은 runner.go 가 연다. 섞으면 「claim.go 가 안
// 채웠다」는 사실이 안 보인다.
//
// 걸음 여덟이고 순서가 뜻을 가진다:
//
//	0  팩 오류면 거절한다.  요청 여부와 무관하다 — 팩을 적은 것은 계약이고
//	   그 팩이 안 열리는데 조용히 도는 것이 「없음이 실패보다 나쁘다」다
//	1  팩을 c.Pack 에 담는다.  2 보다 앞이라 요청이 0 이어도 스킬은 펴진다 —
//	   계약이 서버를 안 적고 스킬만 쓰는 것이 정상이다
//	2  요청 집합을 만든다.  비면 여기서 반환한다 — 서버 출처를 아예 안 본다
//	3  워크스페이스 파일을 못 읽었으면 거절한다.  4 보다 앞이라 깨진 파일이
//	   「없는 이름」으로 둔갑하지 않는다
//	4  이름 순으로 출처 셋을 뒤진다.  팩을 가장 먼저 본다 — 우선순위가 아니라
//	   노드 선언 충돌을 놓치지 않기 위해서다.  노드를 먼저 보고 집으면 팩이
//	   같은 이름을 실은 사실을 못 본 채로 지나간다
//	5  집은 항목마다 종류를 채운다.  4 보다 뒤라 우리가 채운 것과 출처가 안 섞인다
//	6  요청 안 한 이름을 Notes 에 남긴다.  워크스페이스와 팩 둘 다
//	7  못 찾은 이름이 있으면 거절한다.  맨 뒤라 Notes 가 다 채워진 뒤다
//
// 거절로 끝날 때도 그때까지의 Components 를 함께 돌려준다 — 왜 실패했는지를
// 아는 데 필요한 사실이 실패와 함께 사라지면 안 된다. 부르는 쪽이 Notes 를
// 먼저 찍고 실패를 낸다 (runner.go 의 ②).
//
// 빈 맵을 세워 돌려주는 것은 쓰는 쪽이 nil 과 빈 것을 안 가르게 하려는 것이다.
func resolveComponents(j Job, p packInput) (Components, error) {
	c := Components{Servers: map[string]map[string]any{}}

	// 0 — Notes 가 오류보다 앞이다. 규약 밖 항목의 이름은 팩이 왜 거절됐는지를
	// 아는 재료이고, 그것이 실패와 함께 사라지면 안 된다.
	c.Notes = append(c.Notes, p.Notes...)
	if p.Err != nil {
		return c, p.Err
	}

	// 1 — 담는 것이 요청을 보기 앞이다. 요청이 0 이어도 스킬은 펴진다.
	c.Pack = p.Pack

	// 2 — 요청이 없으면 허용목록이 빈다. 출처가 무엇을 선언했든 그렇다
	// (features.md 3.2 — 요청이 없을 때 0 은 의도다).
	want := wantedMCP(j.Params.MCP)
	if len(want) == 0 {
		return c, nil
	}

	// 3 — 원인이 파일이면 문구도 파일을 가리킨다. 파일 경로는 안 싣는다:
	// 이 문구가 res.Error 로 봉인에 들어가고, 노드의 디렉터리 구조는 계약
	// 작성자가 알 것이 아니다. 파일은 하나뿐이라 이름으로 충분하다.
	if j.WorkspaceMCPErr != nil {
		return c, fmt.Errorf("cannot read workspace .mcp.json: %w", j.WorkspaceMCPErr)
	}

	// 4
	var packMCP map[string]map[string]any
	if p.Pack != nil {
		packMCP = p.Pack.MCP
	}
	var missing []string
	for _, name := range want {
		if e, ok := packMCP[name]; ok {
			// 팩이 노드 선언 이름을 덮으면 거절이다 — 계약이 requires 로
			// 소유자의 선언을 보고 노드를 고른 뒤 자기 팩의 정의로 그 이름을
			// 덮으면, 매칭은 소유자의 값으로 하고 실행은 계약의 값으로 하는
			// 것이 된다. 소유권이 뒤집히는 자리다.
			//
			// 요청된 이름만 본다 — 그 뒤집힘은 이름이 실제로 허용목록에
			// 실릴 때만 열린다. 전부 보면 실행 위험이 0 인 경우까지 단계를
			// 죽여, 노드가 흔한 이름을 선언해 두면 그 이름을 담은 팩이 그
			// 노드에서 전부 막힌다.
			if _, dup := j.NodeMCP[name]; dup {
				return c, fmt.Errorf("pack redefines node-declared mcp server %s", name)
			}
			// 원본을 안 고친다 — 5 가 키를 하나 더하는데 그 맵은 Pack 의 것이다.
			e = copyEntry(e)
			if !hasText(e, "command") && !hasText(e, "url") {
				// 종류를 못 정하는 항목은 하네스가 말없이 버린다. 워크스페이스
				// 출처와 같은 실패이므로 같은 자리에서 같은 꼴로 거절한다.
				return c, fmt.Errorf(
					"pack mcp server %s declares neither command nor url", name)
			}
			if _, dup := j.WorkspaceMCP[name]; dup {
				// 계약이 실어 보낸 것이 가장 재현 가능하다 (decisions.md 2절).
				c.Notes = append(c.Notes, "mcp server "+name+" is declared by both "+
					"the pack and the workspace; the pack wins")
			}
			c.Servers[name] = e
			continue
		}
		if s, ok := j.NodeMCP[name]; ok {
			if _, dup := j.WorkspaceMCP[name]; dup {
				// 노드가 이긴다. 저장소에 쓰는 사람이 소유자가 선언한
				// 이름을 가로채지 못한다 — 매칭은 소유자의 값으로 하고
				// 실행은 저장소의 값으로 하는 길이 닫힌다.
				c.Notes = append(c.Notes, "mcp server "+name+" is declared by both "+
					"the node and the workspace; the node declaration wins")
			}
			c.Servers[name] = s.allowlistEntry()
			continue
		}
		if e, ok := j.WorkspaceMCP[name]; ok {
			// 원본을 안 고친다 — 4 가 키를 하나 더하는데 그 맵은 Job 의 것이다.
			e = copyEntry(e)
			if !hasText(e, "command") && !hasText(e, "url") {
				// 종류를 못 정하는 항목은 하네스가 말없이 버린다. 계약이
				// 이름으로 요청한 것이 조용히 사라지는 것은 아래 6 이 막으려는
				// 바로 그 실패다.
				return c, fmt.Errorf(
					"workspace mcp server %s declares neither command nor url", name)
			}
			c.Servers[name] = e
			continue
		}
		missing = append(missing, name)
	}

	// 5 — 종류를 채우는 자리가 하나다. 출처 셋이 같은 규칙을 받는다.
	for _, e := range c.Servers {
		ensureType(e)
	}

	// 6 — 계약 작성자가 「적어 뒀는데 왜 없나」를 여기서 푼다.
	requested := make(map[string]bool, len(want))
	for _, name := range want {
		requested[name] = true
	}
	for _, name := range entryNames(j.WorkspaceMCP) {
		if !requested[name] {
			c.Notes = append(c.Notes, "workspace .mcp.json declares "+name+
				", which this step did not request")
		}
	}
	// 팩이 실었으나 요청 안 한 이름. 팩은 정의의 출처이지 허가의 출처가 아니다 —
	// 이 줄이 없으면 계약 작성자가 이름을 안 적고도 임의의 stdio 서버를 물릴 수
	// 있고, --strict-mcp-config 도 가짜 홈도 그것을 안 막는다.
	for _, name := range entryNames(packMCP) {
		if !requested[name] {
			c.Notes = append(c.Notes, "pack declares mcp server "+name+
				", which this step did not request")
		}
	}

	// 7 — 조용히 빼고 돌면 하네스가 exit 0 으로 끝나고 성공이 봉인된다.
	// 단계는 초록인데 모델은 그 도구를 못 봤다 (ADR-035 §3).
	if len(missing) > 0 {
		for _, name := range missing[1:] {
			c.Notes = append(c.Notes, notAvailable(name))
		}
		return c, errors.New(notAvailable(missing[0]))
	}
	return c, nil
}

// notAvailable 은 팩이 박은 문구다 (features.md 3.2 · scene-gates.md CA4).
//
// 게이트가 부분 문자열로 찾으므로 이름 하나짜리 단수형을 유지한다. 없는
// 이름이 여럿이면 첫 이름이 문구로 나가고 나머지는 Notes 로 간다.
func notAvailable(name string) string {
	return "mcp server " + name + " is not available on this node"
}

// wantedMCP 는 계약이 요청한 이름을 정렬된 유일한 목록으로 만든다.
//
// 같은 이름을 두 번 적은 계약을 거절하지 않는다 — 이름의 집합이 같으므로
// 결과가 같다. 정렬하는 이유는 거절 문구와 Notes 의 순서가 같은 계약이면
// 같아야 하기 때문이다 (ADR-014 결정 3 의 결).
func wantedMCP(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// entryNames 는 항목 맵의 이름을 정렬해 낸다. 같은 파일이면 같은 순서다.
func entryNames(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// copyEntry 는 항목을 얕게 복사한다.
//
// ensureType 이 맵을 고치는데 그 맵은 Job 의 것이다 — 워크스페이스 파일에서
// 읽은 그대로다. 원본을 고치면 같은 Job 을 두 번 쓰는 호출자와 시험이 조용히
// 갈린다. 값은 안 복사한다 — 우리는 키 하나만 더하고 값은 안 만진다.
func copyEntry(e map[string]any) map[string]any {
	out := make(map[string]any, len(e)+1)
	for k, v := range e {
		out[k] = v
	}
	return out
}

// hasText 는 그 키에 빈 문자열이 아닌 문자열이 있는가다.
//
// 문자열이 아닌 값은 「없다」로 본다 — {"command": 5} 는 종류를 못 정하므로
// 우리가 채울 수도 없고, 사람이 고칠 자리는 같은 줄이다.
func hasText(e map[string]any, key string) bool {
	s, ok := e[key].(string)
	return ok && s != ""
}

// ensureType 은 항목에 종류를 채운다 (features.md 3.2 · decisions.md 6절 ㉓).
//
// 실측이 근거다 — claude 2.1.266 은 type 도 command 도 없는 항목을 init 줄의
// mcp_servers 에서 말없이 뺀다. failed 로도 안 나타난다. ADR-035 §4.4 의
// 예시(url + credential)가 그대로 그 모양이라, 그 선언은 광고는 서고 서버는
// 안 열리는 조합을 조용히 만든다 — ADR-035 §3 의 「없음이 실패보다 나쁘다」다.
//
// 이미 있으면 안 덮는다. 소유자가 type: sse 를 적었거나 저장소가 http 를
// 적었으면 그것이 이긴다 — 우리는 비어 있는 자리만 채운다.
//
// 종류를 잘못 채우는 경우는 남는다 (sse 서버를 url 만으로 적으면 http 가 된다).
// 그때 그 서버는 failed 로 목록에 나타난다 — 침묵이 아니라 실패다.
func ensureType(e map[string]any) {
	if _, ok := e["type"]; ok {
		return
	}
	switch {
	case hasText(e, "command"):
		e["type"] = "stdio"
	case hasText(e, "url"):
		e["type"] = "http"
	}
}

// workspaceMCPName 은 워크스페이스가 하네스에게 쓰는 파일이다 (features.md 3.4).
const workspaceMCPName = ".mcp.json"

// readWorkspaceMCP 는 워크스페이스의 선언을 원문 그대로 읽는다.
//
// 가장자리다 — 정하는 자리가 아니다. 등급도 문구도 resolveComponents 가
// 정하므로 오류를 그대로 올린다.
//
// 원문 그대로인 이유는 이 파일이 우리 형식이 아니기 때문이다. 저장소가
// 하네스에게 쓴 것이고 type · headers 처럼 우리 어휘에 없는 키가 있다.
// 우리는 전달자이고 번역자가 아니다.
//
//	없는 파일               nil.  저장소 대부분에 그 파일이 없다
//	mcpServers 가 없는 파일   nil.  다른 목적의 파일일 수 있다
//	그 밖의 실패             오류.  요청이 있을 때만 만난다 — claim.go 가
//	                       그때만 부른다
//
// 크기 상한을 안 둔다 — 같은 워크스페이스를 collectDeclared 와 워크스페이스
// diff 가 이미 통째로 읽는다. 여기만 상한을 두면 규율이 갈린다. 푼 뒤의
// 크기를 지는 것은 팩이다 (U5).
func readWorkspaceMCP(dir string) (map[string]map[string]any, error) {
	b, err := os.ReadFile(filepath.Join(dir, workspaceMCPName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var f struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.MCPServers, nil
}

// 팩 tar 를 읽는다 — 규약 안과 밖을 접두로 가른다
//
// 규약 안은 셋이다 (decisions.md 2절):
//
//	skills/<이름>/SKILL.md   스킬.  같은 디렉터리의 곁 파일도 함께 옮긴다
//	agents/<이름>.md         서브에이전트
//	mcp.json                 팩의 MCP 정의.  루트의 그 이름 하나
//
// 접두로 가르는 것이 이름 하나를 건너뛰는 것보다 낫다 — 실을 것을 나열하는
// 규칙이라 하네스나 규약이 이름을 늘려도 안 샌다. 이 저장소가 R1 환경
// 화이트리스트와 logs/ 에서 이미 고른 규율이다.

// packMCPName 은 팩 루트의 서버 정의 파일이다.
const packMCPName = "mcp.json"

// packSettingsName 은 팩이 실어서는 안 되는 파일이다 (ADR-034 §7).
//
// 접두 규칙이 이미 막으므로 이 이름이 따로 필요하진 않다. 그래도 두는 이유는
// 문구다 — 훅 설정이 정책이고 팩이 그것을 덮으려 했다는 사실이 로그에 이름으로
// 남아야 한다. 조용한 무시와 보이는 무시를 가른다.
const packSettingsName = "settings.json"

const (
	packSkillsPrefix = "skills/"
	packAgentsPrefix = "agents/"
)

// errPackTooBig 는 받은 바이트가 상한을 넘었다는 사실이다 (R8).
//
// 문구가 아니라 사실인 이유는 되싸기 때문이다 — tar 도 gzip 도 상한에 걸린
// 읽기를 자기 형식의 오류로 되싼다. 사실로 들고 다녀야 overLimit 이 그것을
// 「tar 가 아니다」와 가를 수 있다.
var errPackTooBig = errors.New("pack exceeds the byte limit")

// countingReader 는 흘러간 바이트를 세고 상한을 넘으면 끊는다.
//
// 상한과 같은 크기까지는 통과시킨다 — 넘은 뒤에만 끊으므로 정확히 MaxBytes
// 인 팩이 안 걸린다. 넘침은 버퍼 한 번치(bufio 의 4 KiB)까지 초과할 수 있고,
// 그 여유는 상한의 뜻을 안 바꾼다.
type countingReader struct {
	r   io.Reader
	max int64
	n   int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.max {
		return n, errPackTooBig
	}
	return n, err
}

// overLimit 은 상한을 먼저 본다.
//
// tar 와 gzip 이 상한 오류를 자기 형식의 오류로 되싸므로, 그대로 적으면
// 「tar 가 아니다」가 나가고 사람이 엉뚱한 것을 고친다.
func overLimit(err error, lim PackLimits, alt error) error {
	if errors.Is(err, errPackTooBig) {
		return fmt.Errorf("pack exceeds the size limit of %d bytes", lim.MaxBytes)
	}
	return alt
}

// readPack 은 tar 하나를 검증해 Pack 으로 만든다 (SEC-A · R3 ~ R16).
//
// 순수하다 — io.Reader 하나를 받고 파일을 하나도 안 쓴다. 거부가 파일을
// 남기기 전에 일어나야 하기 때문이고, 그래서 시험이 악성 tar 를 메모리에서
// 지어 넣는다. 여는 것은 runner.go 의 openPack 이다.
//
// 이름을 인자로 받는 이유는 문구다 — R3 과 R16 이 팩 이름을 담는데, 바깥에서
// 감싸면 게이트가 찾는 글자가 그대로 안 나온다.
//
// 걸음 다섯이고 순서가 뜻을 가진다:
//
//	1  받은 바이트를 세면서 sha256 에 흘린다
//	2  첫 두 바이트가 1f 8b 면 gzip 을 한 겹 벗긴다
//	3  항목마다 종류 · 개수 · 이름 · 크기를 보고 가른다.
//	   종류가 이름보다 앞이다 — 심볼릭 링크의 이름이 성해 보일 수 있고,
//	   그때 이름 검사만 통과시키면 링크가 살아 나간다
//	4  남은 바이트를 끝까지 읽어 해시를 마친다.  tar 는 꼬리에 0 블록을 달고
//	   거기서 멈추면 해시가 파일 전체의 값이 아니다
//	5  실을 것이 0 이면 거절한다.  하네스는 없는 플러그인 디렉터리를
//	   종료코드 0 에 stderr 한 줄 없이 무시하므로 빠짐을 우리가 잡는다
//
// Notes 는 오류와 함께도 돌려준다 — 왜 실패했는지를 아는 데 필요한 사실이
// 실패와 함께 사라지면 안 된다.
func readPack(name string, r io.Reader, lim PackLimits) (*Pack, []string, error) {
	sum := sha256.New()
	counted := &countingReader{r: r, max: lim.MaxBytes}
	br := bufio.NewReader(io.TeeReader(counted, sum))

	// 형식이 아닌 모든 실패가 이 문구로 모인다 — 팩을 짓는 사람이 고칠
	// 자리가 하나이기 때문이다 (tar 가 아니거나 잘렸거나 덜 왔다).
	notTar := fmt.Errorf("pack %s is not a tar archive", name)

	var src io.Reader = br
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		// 한 겹만 벗긴다 — 안이 또 gzip 이면 tar 로 안 읽혀 아래가 거절한다.
		zr, err := gzip.NewReader(br)
		if err != nil {
			return nil, nil, overLimit(err, lim, notTar)
		}
		defer zr.Close() //nolint:errcheck
		src = zr
	}

	p := &Pack{}
	var notes []string
	at := map[string]int{}    // 이름 -> Files 의 자리. 뒤가 이긴다 (R15)
	seen := map[string]bool{} // 규약 안에서 같은 이름을 두 번 봤나
	var entries int
	var expanded int64

	tr := tar.NewReader(src)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, notes, overLimit(err, lim, notTar)
		}
		// 종류가 먼저다 (R4). 우리가 필요한 디렉터리는 쓰는 쪽이 직접 만들므로
		// tar 의 디렉터리 항목은 개수만 세고 버린다.
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeDir {
			return nil, notes, fmt.Errorf(
				"pack entry %s is a %s; only regular files are read",
				hdr.Name, packEntryKind(hdr.Typeflag))
		}
		entries++
		if entries > lim.MaxFiles {
			return nil, notes, fmt.Errorf(
				"pack exceeds the file limit of %d entries", lim.MaxFiles)
		}
		clean, err := packEntryName(hdr.Name)
		if err != nil {
			return nil, notes, err
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		// 읽기 전에 센다 — 헤더가 거짓말해도 받은 바이트 쪽 상한이 잡는다.
		expanded += hdr.Size
		if expanded > lim.MaxBytes {
			return nil, notes, fmt.Errorf(
				"pack exceeds the size limit of %d bytes", lim.MaxBytes)
		}

		inside := clean == packMCPName ||
			strings.HasPrefix(clean, packSkillsPrefix) ||
			strings.HasPrefix(clean, packAgentsPrefix)
		if !inside {
			if clean == packSettingsName {
				notes = append(notes, "pack carries settings.json; this run does not read it")
				continue
			}
			notes = append(notes, "pack carries "+clean+
				", which is outside the pack convention; ignoring it")
			continue
		}
		if seen[clean] {
			notes = append(notes, "pack carries "+clean+" more than once; the last one wins")
		}
		seen[clean] = true

		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, notes, overLimit(err, lim, notTar)
		}
		if clean == packMCPName {
			m, err := packMCPServers(b)
			if err != nil {
				return nil, notes, fmt.Errorf("pack mcp.json is not valid json: %v", err)
			}
			p.MCP = m
			continue
		}
		if i, dup := at[clean]; dup {
			p.Files[i].Data = b
			continue
		}
		at[clean] = len(p.Files)
		p.Files = append(p.Files, PackFile{Name: clean, Data: b})
	}

	// 4 — 꼬리까지 읽어야 해시가 파일 전체의 값이다. 빠뜨리면 봉인에 남은
	// 값으로 blob 을 대조할 수 없다.
	if _, err := io.Copy(io.Discard, br); err != nil {
		return nil, notes, overLimit(err, lim, notTar)
	}
	if len(p.Files) == 0 && len(p.MCP) == 0 {
		return nil, notes, fmt.Errorf(
			"pack %s carries no skills, agents, or mcp servers", name)
	}
	sort.Slice(p.Files, func(i, j int) bool { return p.Files[i].Name < p.Files[j].Name })
	p.SHA256 = hex.EncodeToString(sum.Sum(nil))
	return p, notes, nil
}

// packEntryKind 는 규약 밖 종류의 이름이다 (R4).
//
// 문구가 「pack entry %s is a %s」라 이 글자가 곧 사람이 읽는 사유다.
// 모르는 종류에는 tar 의 글자를 그대로 보인다 — 이름을 지어내면 그 이름으로
// 찾을 문서가 없다.
func packEntryKind(flag byte) string {
	switch flag {
	case tar.TypeSymlink:
		return "symlink"
	case tar.TypeLink:
		return "hard link"
	case tar.TypeChar, tar.TypeBlock:
		return "device"
	case tar.TypeFifo:
		return "fifo"
	}
	return fmt.Sprintf("tar type %q", rune(flag))
}

// packEntryName 은 항목 이름을 검증하고 정규화한다 (R5 ~ R7).
//
// 검증이 정규화보다 앞인 것이 규칙이다 — path.Clean 은 .. 를 먹어 치우므로
// 먼저 돌리면 escapes the pack root 가 영영 안 걸린다.
//
// 백슬래시를 거절하는 이유는 크로스 빌드다. tar 의 이름은 / 로 나뉘는데
// 윈도우에서는 \ 도 구분자다. 같은 팩이 리눅스와 윈도우에서 다른 경로로
// 펴지면 격리의 경계가 플랫폼마다 달라진다.
func packEntryName(raw string) (string, error) {
	if strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("pack entry %s has an absolute path", raw)
	}
	for _, part := range strings.Split(raw, "/") {
		if part == ".." {
			return "", fmt.Errorf("pack entry %s escapes the pack root", raw)
		}
	}
	if !packSafeName(raw) {
		return "", fmt.Errorf("pack entry %s has an unsafe name", raw)
	}
	clean := path.Clean(raw)
	if clean == "." || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("pack entry %s has an unsafe name", raw)
	}
	return clean, nil
}

// packSafeName 은 이름에 실어서는 안 되는 글자가 있는가다.
func packSafeName(raw string) bool {
	if raw == "" {
		return false
	}
	if strings.ContainsRune(raw, '\\') {
		return false
	}
	// 드라이브 문자. C:/x 도 C:x 도 윈도우에서는 우리가 지은 뿌리 밖이다.
	if len(raw) >= 2 && raw[1] == ':' {
		return false
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// packMCPServers 는 팩 mcp.json 의 mcpServers 를 원문 그대로 집는다.
//
// mcpServers 키가 없는 파일은 서버 0 이다 — readWorkspaceMCP 와 같은 판단이고,
// 다른 목적의 파일일 수 있다.
func packMCPServers(b []byte) (map[string]map[string]any, error) {
	var f struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.MCPServers, nil
}

// mcpAllowlistName 은 허용목록 파일의 이름이다 (decisions.md 2절).
//
// 가짜 홈 밖에 둔다 — 홈 안에 두면 하네스가 설정으로도 읽을 수 있고,
// 우리가 --mcp-config 로 가리키는 것과 그것이 갈릴 자리가 생긴다.
const mcpAllowlistName = "mcp.json"

// writeMCPAllowlist 는 허용목록 파일을 쓴다 (features.md 3.2).
//
// 권한은 0600 이다 — 이름만 든 파일이지만 계장 안의 다른 파일과 같은 값으로 둔다.
func writeMCPAllowlist(path string, servers map[string]map[string]any) error {
	b, err := mcpAllowlistJSON(servers)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// mcpAllowlistJSON 은 서버 맵을 mcp.json 한 줄로 만든다.
//
// 봉투를 언제나 쓴다 — 빈 것도 {"mcpServers":{}} 이고 {} 가 아니다.
// --strict-mcp-config 는 「이 파일에 적힌 것만」이고 그 「적힌 것」의 자리가
// mcpServers 다. 키가 없는 파일을 하네스가 어떻게 읽는지는 실측이 없고,
// 봉투를 언제나 쓰면 그 물음이 사라진다. 그리고 0 이 의도임이 파일에 보인다
// (features.md 3.2 — 요청이 없을 때 0 은 의도다).
func mcpAllowlistJSON(servers map[string]map[string]any) ([]byte, error) {
	// nil 도 빈 봉투다 — 아래 Marshal 이 nil 맵을 null 로 적는다.
	if servers == nil {
		servers = map[string]map[string]any{}
	}
	b, err := json.Marshal(struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}{servers})
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// allowlistEntry 는 서버 하나를 mcp.json 의 객체로 만든다.
//
// 비면 키를 안 쓴다. command: "" 를 적어 보내면 하네스가 그것을 실행 경로로
// 읽을 수 있고, 그 실패는 우리 파일이 만든 것이 된다. 빈 값은 「안 적었다」이지
// 「빈 것을 적었다」가 아니다.
//
// Credential 은 안 나간다 — 하네스가 모르는 키이고(우리 어휘다), remote 인증은
// 노드 환경변수로 가므로 이름을 파일에 적어도 하네스가 그것으로 하는 일이 없다.
// 그 이름이 사는 자리는 mcpUp 의 remote 판정 하나다.
//
// Extra 를 먼저 얹고 아는 키를 그 위에 덮는다. 아는 키가 이겨야 소유자가 적은
// 선언과 우리가 읽은 선언이 안 갈린다 (decisions.md 2절).
func (s MCPServer) allowlistEntry() map[string]any {
	e := map[string]any{}
	for k, v := range s.Extra {
		e[k] = v
	}
	if s.Command != "" {
		e["command"] = s.Command
	}
	if len(s.Args) > 0 {
		e["args"] = s.Args
	}
	if s.URL != "" {
		e["url"] = s.URL
	}
	if len(s.Env) > 0 {
		e["env"] = envRefs(s.Env)
	}
	return e
}

// envRefs 는 이름을 참조로 바꾼다 (features.md 3.2).
//
// 값을 적지 않는다 — 값은 R1 화이트리스트가 노드 환경으로 넘기고 하네스가
// 그 참조를 편다. validateEnvNames 가 값 자리를 환경변수 이름으로 보장하므로
// 여기서 다시 안 본다.
func envRefs(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = "${" + v + "}"
	}
	return out
}
