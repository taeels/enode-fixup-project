# U5 `pack` — 팩의 형식과 기록의 자리

`features.md` 3.6 · 3.7 의 형식이다. 규칙은 `business-rules.md`, 지나가는 길은
`business-logic-model.md`.

답 여덟은 전부 A 다 (계획 2절 · 사용자 결정 2026-09-14). 그중 셋이 이 문서의
형식을 정했다 — 답 1 이 팩이 사는 자리를, 답 2 가 팩 `mcp.json` 의 어휘를,
답 4 가 상한의 값을 정했다.

---

## 1. 팩은 tar 하나다 — 규약 안과 밖

`decisions.md` 2절이 규약을 이미 닫았다. 이 문서는 그 규약을 **읽는 쪽의 형식**으로
옮긴다.

```text
   규약 안    skills/<이름>/SKILL.md      스킬.  디렉터리째 옮긴다
              skills/<이름>/<그 밖>       같은 디렉터리의 곁 파일도 옮긴다
              agents/<이름>.md            서브에이전트
              mcp.json                    팩의 MCP 정의.  루트의 그 이름 하나

   규약 밖    그 밖의 모든 항목            무시한다.  이름을 Notes 로 남긴다
              settings.json               무시한다.  Notes 의 문구가 다르다 —
                                          훅 설정이 정책이고 팩이 그것을
                                          덮으면 안 된다 (ADR-034 §7)
```

**접두로 가른다** — `skills/` 와 `agents/` 로 시작하는 항목만 싣는다. 이름 하나를
건너뛰는 규칙(`settings.json`)이 아니라 **실을 것을 나열하는 규칙**이다. 이 저장소가
같은 물음에서 이미 고른 규율이다 (R1 환경 화이트리스트 · `logs/` 허용목록 ⑱).

**압축은 받는다** (답 5=A). 첫 두 바이트가 `1f 8b` 이면 gzip 을 한 겹 벗기고 그
안을 tar 로 읽는다. 계약 저자가 쓸 수 있는 것이 argv 하나뿐이라 `.tar.gz` 를
받아 다시 푸는 자리가 계약에 없기 때문이다.

---

## 2. `Pack` · `PackFile` — 검증을 통과한 것만 담는다

```go
// Pack 은 검증을 통과한 팩이다. tar 를 다시 안 연다.
//
// 파일이 아니라 메모리다 — 거부가 파일을 남기기 전에 일어나야 하기 때문이다
// (SEC-A · decisions.md 6절 ②). 쓰는 쪽(Instrument)은 검증을 다시 안 한다.
type Pack struct {
	// SHA256 은 $IN 에서 받은 바이트의 것이다. 압축된 원본 그대로다.
	SHA256 string

	// Files 는 규약 안의 항목이다. 이름 순으로 담는다.
	Files []PackFile

	// MCP 는 팩 mcp.json 의 mcpServers 를 원문 그대로 담는다 (답 2=A).
	MCP map[string]map[string]any
}

// PackFile 은 펴질 파일 하나다.
type PackFile struct {
	Name string // 팩 안의 상대경로. skills/ 또는 agents/ 로 시작한다
	Data []byte
}
```

### 2.1 `Mode` 를 안 싣는다 — 설계 원안에서 뺀 필드다

`component-methods.md` 2.1 은 `PackFile` 에 `Mode fs.FileMode` 를 두었다. **뺀다.**

```text
   규약 안의 파일이 전부 글자다      SKILL.md 와 <이름>.md 다.  실행 비트가 할 일이 0
   tar 의 모드를 그대로 쓰면         팩이 0777 파일을 계장 안에 남긴다.
                                    받을 이유가 없는 값을 받는 것이다
   그래서                           파일은 0600 · 디렉터리는 0700 으로 우리가 정한다.
                                    계장 안의 다른 파일과 같은 값이다
                                    (mcp.json · .credentials.json 이 0600)
```

### 2.2 `SHA256` 은 받은 바이트의 것이다

**푼 뒤가 아니라 받은 그대로다.** 봉인에 남는 값이 계약이 나른 blob 의 신원이어야
하기 때문이다 — 기록을 읽는 사람이 같은 Run 의 blob 을 받아 `sha256sum` 으로
그 값과 맞출 수 있어야 한다. 푼 바이트로 재면 그 대조가 안 선다.

`gzip` 이면 압축된 바이트의 값이다. tar 가 끝난 뒤에도 **남은 바이트를 끝까지
읽어** 해시를 마친다 — tar 는 꼬리에 0 블록을 달고, 거기서 멈추면 파일 전체의
값이 아닌 것이 나온다.

접두를 안 붙인다. 이 저장소의 다른 해시가 전부 맨 hex 다 (`identity.go:92` ·
`demo_gallery.go:83`).

---

## 3. `PackLimits` — 값 둘 (답 4=A)

```go
// PackLimits 는 푸는 쪽이 디스크와 메모리를 채우는 길을 막는다 (SEC-A).
type PackLimits struct {
	MaxBytes int64 // 64 MiB. 받은 바이트에도 푼 바이트에도 같이 건다
	MaxFiles int   // 512. 규약 밖 항목까지 함께 센다
}

var defaultPackLimits = PackLimits{MaxBytes: 64 << 20, MaxFiles: 512}
```

```text
   왜 64 MiB    전송은 이미 막혀 있다 — Mediator 의 MaxBlobBytes 기본 10 MiB 가
                tar 자체를 막는다.  그 여섯 배를 푼 바이트에 준다.  gzip 을
                받으므로(답 5=A) 압축률 6.4 까지가 그 안이고, 글자 팩의 실제
                압축률이 거기 못 미친다 — 폭탄만 걸리고 성한 팩은 안 걸린다
   왜 한 값이 둘을 지나  받은 바이트와 푼 바이트 중 먼저 닿는 쪽에서 끊는다.
                값이 하나라 「무엇을 넘었나」를 사람이 안 헷갈린다
   왜 512       스킬 팩의 실제 파일 수는 열 단위다.  사고를 잡는 값이지
                사람을 막는 값이 아니다
   파일 하나의 상한  안 둔다.  합계가 이미 그것을 진다
```

---

## 4. 팩의 `mcp.json` 은 원문 그대로다 (답 2=A)

```json
{"mcpServers": {"probe4": {"command": "true"}}}
```

`Pack.MCP` 의 타입이 `map[string]map[string]any` 인 이유는 U4 가 워크스페이스
`.mcp.json` 에서 고른 것과 같다 — **이 파일은 하네스가 정의한 형식**이고 우리
어휘가 아니다.

```text
   MCPServer 로 받으면   type · headers 처럼 우리가 모르는 키가 사라진다
                        (Extra 는 UnmarshalYAML 이 채우므로 JSON 경로에 없다)
                        env 의 뜻이 뒤집힌다 — 노드 선언은 이름에서 이름으로 가고
                        이 파일의 env 는 값이다.  envRefs 가 ${${X}} 를 만든다
   원문으로 받으면       번역하는 코드가 없다.  번역이 못 틀린다
```

`component-methods.md` 2.1 의 `Pack.MCP map[string]MCPServer` 한 줄이 이 답으로
낡는다 — 이 유닛의 파장이다.

---

## 5. `Components` 가 자라는 자리

U1 이 세우고 U4 가 채운 그릇에 팩이 앉는다. **형식은 안 바뀐다.**

```go
type Components struct {
	Servers map[string]map[string]any // 이름 -> 허용목록 항목. 출처 셋이 여기로 모인다
	Pack    *Pack                     // nil 이면 팩 없음. U5 가 채운다
	Notes   []string                  // 로그로 낼 사실
}
```

**`Servers` 와 `Pack` 이 서로 다른 것을 진다.** 팩의 서버는 `Servers` 로 들어가
`agent.mcp` 필터를 타고(⑥), 팩의 스킬과 서브에이전트는 `Pack.Files` 로 남아
필터 없이 펴진다. 스킬에는 요청 문법이 없기 때문이다 — 계약이 이름으로 고르는
것은 서버뿐이다.

### 5.1 가장자리가 낸 것을 담는 그릇

```go
// packInput 은 가장자리가 연 결과다 (답 3=A).
//
// runHarness 가 채우고 resolveComponents 가 등급과 문구를 정한다. 필드로
// Job 에 안 넣는다 — Job 은 claim.go 가 채우는 것이고 팩은 runner.go 가 연다.
// 섞으면 「claim.go 가 안 채웠다」는 사실이 안 보인다.
type packInput struct {
	Pack  *Pack
	Notes []string // 규약 밖 항목의 이름 등. readPack 이 낸다
	Err   error    // 못 열었거나 검증에 걸렸다. 문구는 resolveComponents 가 짓는다
}
```

---

## 6. `HarnessResult` 의 새 필드 둘 (3.7)

```go
type HarnessResult struct {
	// ... 오늘의 여섯 그대로 ...

	// MCP 는 허용목록에 실제로 실린 서버 이름이다 (ADR-005 성질 4).
	// 요청한 것이 아니라 실린 것이다 — 둘이 갈리면 봉인이 그것을 안다.
	// 이름 순이다.
	MCP []string `json:"mcp,omitempty"`

	// Pack 은 실린 팩 tar 의 sha256 이다. 팩이 없으면 빈 값이다.
	Pack string `json:"pack,omitempty"`
}
```

**`omitempty` 다.** 팩도 서버도 없는 단계가 오늘의 봉인과 한 글자도 안 달라진다 —
`steps/NN-*.json` 을 읽는 사람과 도구가 새 키를 안 만난다.

**둘 다 exec 한 경로에서만 찬다** (답 7=A). exec 앞에서 죽은 단계는 아무것도 안
물렸으므로 빈 값이 참이다. 실패 이유는 `Message` 가 진다.

`runner.go` 의 ⑧ 자리에서 채운다 — `Decode` 가 낸 `HarnessResult` 에 얹는다.
어댑터는 이 값을 모른다. **팩과 허용목록은 어댑터가 정하는 것이 아니라 우리가
정해서 준 것**이므로, 하네스의 대답이 아니라 우리 사실이다.

---

## 7. 계장 아래의 자리 (답 1=A)

```text
   <계장 tmp>/home/                가짜 홈.  U1
   <계장 tmp>/home/settings.json   훅 설정.  U1
   <계장 tmp>/home/.credentials.json
   <계장 tmp>/mcp.json             허용목록.  U1 · U4
   <계장 tmp>/pack/                팩.  U5 가 짓는다 — 홈 밖이다
   <계장 tmp>/pack/skills/…
   <계장 tmp>/pack/agents/…
```

**홈 밖에 두는 것이 답 1=A 의 값이다.** 팩은 `--plugin-dir <계장 tmp>/pack` 으로
들어간다.

```text
   디렉터리 이름이 곧 접두다   pack 이라 스킬 이름이 pack:<이름> 으로 보인다.
                              packDirName 상수 하나가 그 값을 진다
   홈 밖이라 얻는 것          팩이 settings.json 이나 .credentials.json 과
                              같은 나무에 없다.  접두 규칙이 이미 막지만
                              여기서는 규칙이 아니라 구조가 막는다
   계장과 함께 지워진다        runHarness 의 defer os.RemoveAll(tmp) 하나가
                              가짜 홈 · 허용목록 · 팩을 같이 지운다
```

---

## 8. 오케스트레이터 설정 (답 8=A)

```go
type ExecutorConfig struct {
	As    string            `yaml:"as"`
	Attrs map[string]string `yaml:"attrs"`

	// Pack 은 이 함대가 에이전트 단계에 실어 보낼 팩이다 (ADR-034 §2.2).
	// 비면 오늘 그대로 — 팩 단계가 안 붙고 agent.pack 도 안 붙는다.
	Pack *PackConfig `yaml:"pack,omitempty"`
}

type PackConfig struct {
	// Fetch 는 팩 단계의 argv 다. 셸이 없으므로 argv 하나로 끝나야 한다.
	//   ["curl", "-o", "$OUT/pack", "<tar 주소>"]
	//   ["git", "archive", "--remote=<url>", "-o", "$OUT/pack", "<rev>"]
	Fetch []string `yaml:"fetch"`
	// Name 은 그 단계가 내는 blob 이름이다. 비면 "pack".
	// agent.pack 과 in.from 이 같은 이름을 가리킨다.
	Name string `yaml:"name"`
}
```

```text
   비면          오늘 그대로다.  BuildContract 가 단계를 안 더하고
                 adapter_test.go 의 steps[0]=plan · steps[1]=gate 가 그대로 초록이다
   Fetch 가 비면  LoadConfig 가 거절한다.  pack: 을 적었는데 argv 가 없는 것은
                 설정 오류이지 「팩 없음」이 아니다
   Name 이 비면   pack 이다
```

---

## 9. 이 유닛이 만들지 않는 이름

```text
   agent.pack_sha256 류의 기대 다이제스트   이월 (decisions.md 6절 ③)
   runctl submit --pack                    이월 (features.md 4절)
   팩 캐시 · 팩 저장소 어휘                 이월
   계약 어휘의 새 키                        0.  U2 가 닫았다 (6절 ⑫)
   새 라우트 · 새 전송                      0.  팩은 있는 blob 경로로만 흐른다
```
