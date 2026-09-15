# U4 `sources` — Code Generation 계획

**유닛** `sources` · **브랜치** `unit/sources` · **담당** taeels ·
**닫는 게이트** CA4 (+ CA0) · **선행** U1 `isolation` · U2 `contract-vocab` ·
U3 `advert`

**이 계획이 Code Generation 의 정본이다.** 여기 없는 것은 안 짓는다.
설계는 `construction/sources/functional-design/` 의 셋이고 규칙 번호 R1 ~ R14 는
그 `business-rules.md` 다.

---

## 0. 앞 단계의 승인과 이 단계가 싣는 것

사용자가 「승인, 코드 써」로 Functional Design 을 닫았다 (커밋 `ab2d669`).

**승인이 열어 준 문서를 이 단계가 싣는다** — U1 이 ⑳ 에서, U2 가 ㉑ 에서,
U3 가 ㉒ 에서 밟은 순서 그대로다.

```text
   decisions.md 6절   실측 행 ㉓.  넷을 한 행에 — 워크스페이스 파일의 어휘 ·
                      url 만 적힌 항목의 침묵 · ${} 가 한 겹만 펴진다 ·
                      모르는 키는 항목을 안 죽인다
   decisions.md 2절   허용목록 행에 「종류를 우리가 채운다」 한 줄 (답 3=A)
   features.md 3.2    파일에 적는 것 목록에 종류(type)가 빠져 있다.  그 줄을 넓힌다
   파일 행렬          2.1 의 U4 칸 · 3절의 빨개지는 옛 시험 · 새 시험 파일 행
```

`scene-gates.md` 는 **안 고친다** — 답 어느 것도 CA4 의 명령을 안 바꾼다.
`unit-of-work.md` 도 안 고친다 — 완료 조건 다섯이 그대로 참이다.

---

## 1. 이 유닛이 내는 diff 의 모양

```text
   제품 파일 셋     행렬 그대로.  새 제품 파일 0
   시험 파일        새 파일 하나(resolve_test.go)와 기존 둘에 붙인다.
                   기존 시험 셋이 타입 변경으로 확정 빨강이라 함께 고친다
   문서            행렬 하나 · 팩 셋 · code-summary.md · 상태와 감사
   패키지 하나      internal/enode.  cmd/ 와 internal/api 는 소스 diff 0
```

| 파일 | 무엇 | 규칙 |
|---|---|---|
| `internal/enode/mcp.go` | `Components.Servers` 타입 · `readWorkspaceMCP` · `resolveComponents` 몸통 · `ensureType` | R1 ~ R11 |
| `internal/enode/runner.go` | `Job` 의 필드 넷 · Notes 로그 한 줄 | R10 · R11 |
| `internal/enode/claim.go` | 읽는 조건과 `Job` 리터럴 네 줄 | R2 · R6 |

**`claude.go` 는 소스 diff 가 0 이다.** `Instrument` 가 `c.Servers` 를
`writeMCPAllowlist` 에 그대로 넘기므로 타입이 바뀌어도 그 줄이 안 바뀐다.
행렬 2.3 의 「U1 · U5 만 만진다」가 그대로 참이다.

**이 유닛이 닫는 스토리** — `unit-of-work-story-map.md` 의 CA4 조각.
계약이 요청한 것만 열리고, 없는 이름은 **조용히 안 빠진다.**

---

## 2. 단계 — 열

### Step 1 — `internal/enode/mcp.go` — 형식

- [x] `Components.Servers` 를 `map[string]map[string]any` 로 올린다.
      주석에 **왜**를 적는다 — 출처마다 어휘가 다르므로 한 형식으로 둘을 못 담고,
      최종 항목으로 올리면 합치는 자리에서 어휘가 하나가 된다
      (`domain-entities.md` 4절)
- [x] `mcpAllowlistJSON` 이 항목을 **짓지 않고 받는다.** 오늘 안에서 부르는
      `s.allowlistEntry()` 가 걷히고 받은 맵을 그대로 봉투에 싣는다
- [x] `writeMCPAllowlist` 의 인자 타입도 같이 바뀐다. 권한 `0600` 은 그대로
- [x] `allowlistEntry()` 는 **안 고친다.** 노드 선언을 항목으로 바꾸는 함수이고
      그 일이 안 바뀌었다. 종류를 채우는 것은 Step 3 의 공용 자리다

### Step 2 — `internal/enode/mcp.go` — 읽기

- [x] `const workspaceMCPName = ".mcp.json"`
- [x] `readWorkspaceMCP(dir string) (map[string]map[string]any, error)`
      — `<dir>/.mcp.json` 을 읽어 `mcpServers` 아래를 그대로 돌려준다
- [x] 없는 파일은 `nil, nil` 이다 (`errors.Is(err, fs.ErrNotExist)`).
      **오류가 아니다** — 저장소 대부분에 그 파일이 없다
- [x] `mcpServers` 키가 없는 파일도 `nil, nil` 이다. 다른 목적의 파일일 수 있다
- [x] JSON 이 아니거나 종류가 틀리면 오류를 그대로 돌려준다. **문구를 여기서
      안 짓는다** — 등급과 문구는 `resolveComponents` 가 정한다 (R7)
- [x] 크기 상한을 안 둔다. 4절 ① 이 근거다
- [x] 주석에 이 함수가 **가장자리**임을 적는다 — 정하는 자리가 아니다

### Step 3 — `internal/enode/mcp.go` — 정하기

- [x] `resolveComponents` 의 몸통을 `business-logic-model.md` 3절의 **걸음
      여섯 그대로** 짓는다. 순서가 뜻을 가지므로 주석이 그 순서의 이유를 적는다
- [x] 걸음 1 — `j.Params.MCP` 를 집합으로 만들고 이름 순으로 정렬한다.
      비면 빈 `Components` 를 돌려준다 (R2). 같은 이름이 두 번 적혀도 한 번이다
- [x] 걸음 2 — `j.WorkspaceMCPErr` 가 있으면 거절한다 (R7)
      `cannot read workspace .mcp.json: %w`. **파일 경로를 안 싣는다**
- [x] 걸음 3 — 이름 순으로 돈다. 노드가 먼저다 (R3). 노드에 있고 워크스페이스에도
      있으면 Note 를 남긴다
- [x] 걸음 3b — 워크스페이스 항목은 **얕은 복사** 뒤에 쓴다 (4절 ②).
      `command` 도 `url` 도 비어 있지 않은 문자열이 아니면 거절한다 (R8)
      `workspace mcp server %s declares neither command nor url`
- [x] 걸음 4 — `ensureType(e map[string]any)` 가 종류를 채운다 (R9).
      `type` 이 이미 있으면 손대지 않고, `command` 면 `"stdio"`, `url` 이면
      `"http"` 다. **노드 항목과 워크스페이스 항목이 같은 함수를 지난다**
- [x] 걸음 5 — 워크스페이스가 선언했으나 요청 안 한 이름을 Note 에 담는다 (R10)
      `workspace .mcp.json declares %s, which this step did not request`
- [x] 걸음 6 — 못 찾은 이름이 있으면 **정렬한 첫 이름**으로 거절하고 나머지를
      Note 에 담는다 (R4 · R5)
      `mcp server %s is not available on this node`
- [x] 거절로 끝나는 모든 자리가 **그때까지의 `Components` 를 함께 돌려준다** (R11)
- [x] U1 이 머리 주석에 적은 「출처 셋을 합치는 것은 U4 이고 팩은 U5 다」를
      이 유닛이 한 자리를 지웠음에 맞춰 고친다. 남는 것은 팩이다

### Step 4 — `internal/enode/runner.go`

- [x] `Job` 에 넷을 더한다 — `NodeMCP` · `WorkspaceMCP` · `WorkspaceMCPErr` ·
      `Log`. 주석은 `domain-entities.md` 5절의 문장이다
- [x] `WorkspaceMCPErr` 의 주석이 **왜 필드인가**를 적는다 — 등급을 정하는 것이
      정책이고 정책은 `resolveComponents` 에 있다. 읽는 쪽이 등급까지 정하면
      거절이 `HarnessResult` 를 안 타고 나가서 단계 오류의 꼴이 경로마다 달라진다
- [x] ② 뒤에 Notes 를 찍는다. **오류 검사보다 앞이다** (R11)
      `j.Log.Info("mcp allowlist note", "note", n)` · `Log` 가 nil 이면 안 찍는다
- [x] ②와 ③ 사이라는 것을 함수 머리 주석의 아홉 줄에 반영한다

### Step 5 — `internal/enode/claim.go`

- [x] `Job` 리터럴 앞에서 워크스페이스를 읽는다.
      **조건 둘** — `len(p.MCP) > 0` (R2) 이고 `w.Local.Workspace != ""` (R6)
- [x] `dir` 이 아니라 `w.Local.Workspace` 를 넘긴다. `dir` 은 워크스페이스가
      없을 때 `os.TempDir()` 로 떨어지고, 거기의 `.mcp.json` 은 아무나 쓸 수 있다
- [x] `Job` 리터럴에 네 줄 — `NodeMCP: w.Local.MCP` · `WorkspaceMCP: wsMCP` ·
      `WorkspaceMCPErr: wsErr` · `Log: log`
- [x] 읽기 실패를 여기서 안 다룬다. 오류를 그대로 실어 보낸다

### Step 6 — 실측 하나 (우리가 쓴 파일을 하네스가 어떻게 읽나)

- [x] 시험 안에서 `Job` 둘을 `resolveComponents` 에 넣고 `writeMCPAllowlist` 로
      파일을 쓴다 — 노드 선언 하나(`probe`)와 워크스페이스 하나(`probe3`)
- [x] 그 파일을 `claude --strict-mcp-config --mcp-config=<경로>` 에 물려
      `init` 줄의 `mcp_servers` 를 읽는다. **함대 없이 실물 하네스로 잰다**
- [x] 재는 것은 **이름이 목록에 있는가** 하나다 (`scene-gates.md` 2절 머리).
      가짜 서버라 `failed` 이고 그것이 판정 재료다
- [x] 결과를 `code-summary.md` 와 `decisions.md` ㉓ 에 적는다. **안 나오면**
      R9 의 값이 틀린 것이므로 그 자리에서 멈추고 사용자에게 묻는다

### Step 7 — 시험

- [x] 새 파일 `internal/enode/resolve_test.go`
- [x] R1 · R2 — 요청이 0 이면 `Servers` 가 비고 Notes 도 0 이다
- [x] R3 — 같은 이름이 둘에 있으면 **노드 항목**이 실린다 (`command` 로 잰다).
      Note 가 남는다
- [x] R4 · R5 — 없는 이름 둘이면 오류가 **정렬한 첫 이름**을 들고, 둘째는 Note 다
- [x] R7 — `WorkspaceMCPErr` 를 준 `Job` 이 그 문구로 거절된다.
      **요청이 0 이면 같은 `Job` 이 안 죽는다** (걸음 1 이 먼저다)
- [x] R8 — `command` 도 `url` 도 없는 워크스페이스 항목이 요청되면 거절이다.
      `command: 5` 도 같은 문구다 (4절 ⑤)
- [x] R9 — 넷을 잰다: 노드 stdio 는 `stdio` · 노드 remote 는 `http` ·
      `Extra` 가 `type: sse` 면 그대로 · 워크스페이스 항목의 `type` 과 `headers` 가
      **한 글자도 안 바뀐다**
- [x] R10 — 요청 안 한 워크스페이스 이름이 Note 로 남는다
- [x] R11 — 거절로 끝나는 `Job` 이 **Notes 가 채워진 `Components`** 를 함께 낸다
- [x] 원본 불변 — `resolveComponents` 뒤에 `j.WorkspaceMCP` 의 맵에 `type` 이
      안 생긴다 (4절 ②)
- [x] `readWorkspaceMCP` 의 넷 — 없는 파일 · `mcpServers` 없는 파일 · 깨진 파일 ·
      성한 파일. `t.TempDir()` 하나면 된다
- [x] 배선 — 스텁 하네스를 쓰는 워커 시험으로 **없는 이름을 요청한 단계가
      하네스를 안 띄우고** `res.Error` 에 문구가 드는 것을 잰다.
      스텁이 파일을 하나 떨구게 해 「안 떴다」를 파일 없음으로 잰다.
      U1 의 구멍(`logs_test.go` 가 함수만 부르고 배선을 안 쟀다)이 여기서 안 생기게
- [x] 고치는 옛 시험 셋 — `mcp_test.go` 의 `mcpAllowlistJSON` 둘과
      `resolveComponents(Job{})` 하나. 타입이 바뀌어 확정 빨강이다

### Step 8 — 변이 여섯

넣고 **빨개지는지** 본다. 안 빨개지면 그 규칙을 재는 시험이 없는 것이다.

- [x] ① R3 의 우선순위를 뒤집는다 (워크스페이스가 이긴다)
- [x] ② `ensureType` 을 걷는다
- [x] ③ 걸음 6 의 거절을 걷고 조용히 뺀다
- [x] ④ R8 의 검사를 걷는다
- [x] ⑤ 요청 필터를 걷고 출처의 것을 전부 싣는다
- [x] ⑥ 얕은 복사를 걷고 원본에 `type` 을 박는다

### Step 9 — 게이트 CA0

- [x] `eval "$(scripts/testdb.sh)"` 뒤 `go test ./...` 전부 초록
- [x] 커버리지 — 패키지별 80% 미달 0. `internal/enode` 를 특히 본다 (U3 뒤 86.0%)
- [x] `go vet ./...` · `gofmt -l` 이 빈다 · `go run ./scripts/glyphscan.go`
- [x] U+2605 전수 grep 0 · 크로스 빌드 셋 · 심볼 상한
- [x] 라우트 26 — `grep -rho 'mux\.HandleFunc\|mux\.Handle(' internal/api/*.go | wc -l`
- [x] 워킹트리 청결. 커버리지 실행이 바꾸는 `cmd/enodectl/probe.lock` 은 되돌린다

### Step 10 — 문서와 팩

- [x] `decisions.md` 6절에 실측 행 ㉓ (0절의 넷)
- [x] `decisions.md` 2절 허용목록 행에 종류 한 줄
- [x] `features.md` 3.2 의 「파일에 적는 것」에 종류를 더한다
- [x] 파일 행렬 — 2.1 의 U4 칸 · 3절의 옛 시험 셋 · 새 시험 파일 행
- [x] `construction/sources/code/code-summary.md`
- [x] `domain-entities.md` 2.1 의 한 줄을 고친다 (4절 ③)
- [x] `aidlc-state.md` 의 U4 절과 `audit.md`
- [x] 계획의 체크박스를 전부 `[x]` 로

---

## 3. 갈래를 안 나눈다

제품 파일 셋이 **한 타입 변경을 함께 받는다.** `Components.Servers` 가 바뀌면
`mcp.go` 와 그 시험이 같은 시점에 컴파일되고, `Job` 의 필드가 없으면
`resolveComponents` 의 몸통이 안 선다. 갈래마다 컴파일되는 시점이 하나뿐이라
나누는 비용이 이득보다 크다 — U1 · U2 · U3 과 같은 판정이다.

---

## 4. 이 계획이 정하는 것 — FD 에 없던 자리 다섯

```text
   ①  워크스페이스 파일에 크기 상한을 안 둔다
      그 파일은 노드가 이미 clone 해 디스크에 둔 것이고, 같은 워크스페이스를
      collectDeclared 와 diff 가 이미 통째로 읽는다.  여기만 상한을 두면
      규율이 갈린다.  푼 뒤의 크기를 지는 것은 U5 의 팩이다 (tar bomb)

   ②  워크스페이스 항목은 얕은 복사 뒤에 쓴다
      ensureType 이 맵을 고치는데 그 맵은 Job 의 것이다.  원본을 고치면
      같은 Job 을 두 번 쓰는 시험과 호출자가 조용히 갈린다.  시험이 잰다

   ③  종류를 채우는 자리는 allowlistEntry 가 아니라 공용 헬퍼다
      두 출처가 같은 규칙을 받아야 하므로 합친 뒤에 한 번 지난다.
      domain-entities.md 2.1 의 「allowlistEntry 가 내는 것에 type 한 줄이
      는다」를 그 사실로 고친다 — 최종 항목에 type 이 있는 것은 그대로 참이다

   ④  Notes 는 Info 로 찍고 메시지는 고정이다
      log.Info("mcp allowlist note", "note", n).  문장을 메시지 자리에 넣으면
      로그 집계가 문장마다 갈린다 — 이 저장소가 Warn 에서 이미 고른 꼴이다

   ⑤  R8 은 문자열이 아닌 command · url 을 「없다」로 본다
      {"command": 5} 는 종류를 못 정하므로 같은 거절이다.  문구를 따로 안 만든다 —
      사람이 고칠 자리가 같다 (그 항목의 command 줄)
```

---

## 5. 이 단계가 안 하는 것

```text
   팩 출처         U5.  Pack 은 빈 구조체 그대로 둔다
   HarnessResult   U5.  기록은 이 유닛의 것이 아니다
   광고            detect.go 를 안 만진다.  워크스페이스는 광고에 안 간다 (R12)
   원격 인증        답 7=A.  credential 은 파일에 안 나간다.  한계를 문서가 진다
   새 Reason       안 만든다.  거절은 ReasonError 로 간다
   CA4 전체        6절
```

---

## 6. CA4 는 사람이 잰다

Step 6 이 재는 것은 **우리가 쓴 파일을 하네스가 어떻게 읽나**이고, CA4 는
**노드와 Mediator 를 지나 그 파일이 실제로 쓰이나**다. 그 셋째 줄(없는 이름)은
Step 7 의 배선 시험이 코드로 닫지만, 앞 두 줄은 실제 함대에서 `runctl record`
로 푼 `logs/` 의 첫 줄을 읽어야 초록이다 (`scene-gates.md` 3절).

**집행자는 이 유닛을 구현하지 않은 사람이다** (`scene-gates.md` 2절 머리).
**눈 검증을 보류로 안 넘긴다** (같은 문서 4절).
