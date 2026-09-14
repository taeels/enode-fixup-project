# U5 `pack` — Code Generation 계획

**유닛** `pack` · **브랜치** `unit/pack` · **담당** taeels ·
**닫는 게이트** CA5 · CA6 (+ CA0) · **선행** U1 `isolation` · U2 `contract-vocab` ·
U4 `sources` (셋 다 `main` 에 있다 · PR #33 ~ #36)

**이 계획이 Code Generation 의 정본이다.** 여기 없는 것은 안 짓는다.
설계는 `construction/pack/functional-design/` 의 셋이고 규칙 번호 R1 ~ R35 는
그 `business-rules.md` 다.

---

## 0. 앞 단계의 승인과 이 단계가 싣는 것

사용자가 「64메가 이외 모두 a로 한다」로 Functional Design 을 닫았다 (커밋 `cb93a0d`).

**승인이 열어 준 문서를 이 단계가 싣는다** — U1 이 ⑳ 에서, U2 가 ㉑ 에서,
U3 가 ㉒ 에서, U4 가 ㉓ 에서 밟은 순서 그대로다. 이 유닛의 파장은 FD 계획 5절이
이미 나열했고 Step 13 이 그것을 집행한다.

```text
   decisions.md 6절    실측 행 ㉔.  FD 계획 1.1 ~ 1.5 를 한 행에 —
                       오늘 플래그로는 팩이 안 읽힌다는 사실과 길 둘의 실측값
   decisions.md 2절    워크스페이스 .claude/skills 행이 미정에서 값이 된다
   features.md 5절     미정 둘 중 하나가 빠진다.  같은 이유다
   features.md 3.6     「가짜 홈에 편다」 -> 「계장 아래 팩 디렉터리에 펴고
                       --plugin-dir 로 가리킨다」
   unit-of-work.md 5절 claude.go 줄의 서술
   scene-gates.md 3절  CA5 의 명령 넷.  스킬 이름이 pack:hello · 충돌 줄에
                       agent.mcp 한 줄 · 마지막 줄은 판정이 아니라 확인
   component-methods.md 2.1 의 Pack.MCP 타입 한 줄과 PackFile.Mode
   파일 행렬            U5 열에 새 시험 파일 하나 · 2.2 에 ①.5 한 줄
```

**정본(`enode-design`)은 이 유닛이 안 고친다.** `ADR-034` §1.1 의 팩 규약과
`--plugin-dir` 이 갈리는 자리는 5절 되돌리기 목록에 올릴 값이다.

---

## 1. 이 유닛이 내는 diff 의 모양

```text
   제품 파일 여섯   행렬 그대로.  새 제품 파일 0
   시험 파일        새 파일 하나(pack_test.go)와 기존 넷에 붙인다.
                   resolveComponents 의 인자가 하나 늘어 기존 호출 전부가
                   확정 빨강이라 함께 고친다
   문서            행렬 · 팩 셋 · code-summary.md · 상태와 감사
   패키지 둘        internal/enode · cmd/iapadapter.  internal/api 와
                   internal/contract 는 소스 diff 0
```

| 파일 | 무엇 | 규칙 |
|---|---|---|
| `internal/enode/mcp.go` | `Pack` · `PackFile` · `PackLimits` · `packInput` · `readPack` · `resolveComponents` 몸통 | R3 ~ R22 |
| `internal/enode/runner.go` | `openPack` · ①.5 호출 · ⑧ 의 기록 두 필드 | R1 · R2 · R28 ~ R30 |
| `internal/enode/harness.go` | `HarnessResult` 의 필드 둘 | R28 · R29 |
| `internal/enode/claude.go` | `Instrument` 의 ③ · `packDirName` · `writePack` | R23 ~ R27 |
| `cmd/iapadapter/config.go` | `ExecutorConfig.Pack` · `PackConfig` · `LoadConfig` 검증 | R31 |
| `cmd/iapadapter/contract.go` | `BuildContract` 의 팩 단계 · `planPrompt` 두 줄 | R32 ~ R35 |

**이 유닛이 닫는 스토리** — `unit-of-work-story-map.md` 의 CA5 · CA6 조각.
계약이 실어 보낸 것이 하네스에 실제로 들어가고, **무엇을 실었는지 봉인에 남는다.**

---

## 2. 단계 — 열셋

### Step 1 — `internal/enode/mcp.go` — 형식

- [x] `Pack struct{}` 의 빈 몸을 채운다 — `SHA256 string` · `Files []PackFile` ·
      `MCP map[string]map[string]any` (`domain-entities.md` 2절)
- [x] `PackFile{Name, Data}`. **`Mode` 를 안 만든다** — 규약 안이 전부 글자라
      실행 비트가 할 일이 0 이고, tar 의 모드를 받으면 팩이 0777 파일을 계장에
      남긴다 (2.1)
- [x] `PackLimits{MaxBytes, MaxFiles}` 와 `defaultPackLimits` — 64 MiB · 512.
      주석이 **왜 그 값인가**를 적는다 (3절)
- [x] `packInput{Pack, Notes, Err}`. **`Job` 의 필드가 아니다** — `Job` 은
      `claim.go` 가 채우는 것이고 팩은 `runner.go` 가 연다 (5.1)
- [x] `mcp.go` 머리 주석의 「남은 것은 팩이고 U5 다」를 이 유닛이 한 사실로 고친다

### Step 2 — `internal/enode/mcp.go` — `readPack` (순수)

`readPack(name string, r io.Reader, lim PackLimits) (*Pack, []string, error)`.
걸음은 `business-logic-model.md` 3절 그대로다.

- [x] **이름을 인자로 받는다.** R3 과 R16 의 문구가 팩 이름을 담기 때문이다 —
      바깥에서 감싸면 `pack %s is not a tar archive` 가 글자 그대로 안 나온다.
      `io.Reader` 하나라는 시험의 값은 그대로다 (4절 ①)
- [x] 걸음 1 — 받은 바이트를 세면서 sha256 에 흘린다.
      `raw -> countingReader(MaxBytes) -> io.TeeReader(hash) -> bufio.Reader`
- [x] 걸음 2 — `br.Peek(2)` 가 `1f 8b` 면 `gzip.NewReader` 를 한 겹 벗긴다 (R3).
      **한 겹만이다** — 안이 또 gzip 이면 tar 로 안 읽혀 R3 이 거절한다
- [x] 걸음 3 — `tar.Next` 마다 순서대로 본다. **종류가 이름보다 먼저다** (3절):
      - [x] 종류 (R4) — `TypeReg` 와 `TypeDir` 만 받는다. 그 밖은 거절이고
            문구의 종류 이름을 4절 ② 가 못 박는다
      - [x] 개수 (R9) — 디렉터리와 규약 밖 항목까지 함께 센다
      - [x] 이름 (R5 · R6 · R7) — 절대경로 · `..` · 백슬래시와 제어문자와
            드라이브 문자 · 빈 이름
      - [x] 푼 바이트 (R8) — `hdr.Size` 를 누계에 더하고 넘으면 **읽기 전에**
            끊는다. 헤더가 거짓말해도 받은 바이트 쪽 상한이 그것을 잡는다
      - [x] 가른다 (R11 ~ R15) — `skills/` · `agents/` · 루트 `mcp.json` ·
            그 밖. 같은 이름이 두 번이면 뒤가 이긴다
- [x] 걸음 4 — 남은 바이트를 `io.Copy(io.Discard, br)` 로 끝까지 읽어 해시를
      마친다. **빠뜨리면 안 된다** — tar 꼬리의 0 블록 뒤에서 멈추면 봉인에 남은
      값으로 blob 을 대조할 수 없다 (2.2)
- [x] 걸음 5 — `Files` 를 이름 순으로 정렬하고, 실을 것이 0 이면 거절한다 (R16)
- [x] `mcp.json` 은 `json.Unmarshal` 로 `mcpServers` 아래를 원문 그대로 집는다.
      깨졌으면 거절 (R12) · 키가 없으면 서버 0 (`readWorkspaceMCP` 와 같은 판단)
- [x] 파일을 하나도 안 쓴다 (R10). 시험이 그것을 잰다

### Step 3 — `internal/enode/mcp.go` — `resolveComponents` 에 팩을 더한다

- [x] 시그니처가 `resolveComponents(j Job, p packInput) (Components, error)` 가 된다.
      **`Job` 에 필드로 안 넣는다** (5.1). 호출자는 `runner.go` 하나다
- [x] 몸통을 `business-logic-model.md` 4절의 **걸음 여덟 그대로** 짓는다.
      순서가 뜻을 가지므로 주석이 그 순서의 이유를 적는다
- [x] 걸음 0 — `p.Err` 가 있으면 그대로 거절한다. **감싸지 않는다** — 게이트가
      부분 문자열로 찾는 글자다. **요청 조기 반환보다 앞이다** (4절 0 이 2 앞)
- [x] 걸음 1 — `c.Pack = p.Pack` 이고 `p.Notes` 를 `c.Notes` 로 옮긴다.
      **걸음 2 보다 앞이다** — 요청이 0 이어도 스킬은 펴진다 (CA5 의 둘째 줄)
- [x] 걸음 2 — 요청이 비면 여기서 반환한다. `c.Pack` 은 담긴 채로 나간다
- [x] 걸음 4 — 이름 순 순회에서 **팩을 가장 먼저 본다.** 우선순위가 아니라
      노드 선언 충돌(R19)을 놓치지 않기 위해서다
      - [x] 팩에 있고 노드에도 있으면 거절 — `pack redefines node-declared mcp server %s`
      - [x] 팩에만 있으면 팩 것. 워크스페이스에도 있으면 Note 를 남긴다 (R21)
      - [x] 팩 항목도 **얕은 복사** 뒤에 쓴다. `ensureType` 이 고치는 맵이
            `Pack.MCP` 의 것이라 원본을 고치면 같은 팩을 두 번 쓰는 시험이 갈린다
      - [x] **계획에 없던 규칙 하나를 더했다** — `command` 도 `url` 도 없는 팩
            항목이 요청되면 거절한다 (`pack mcp server %s declares neither
            command nor url`). U4 가 워크스페이스 출처에서 같은 실패를 이미
            거절했고(㉓ ②), 종류를 못 정하는 항목을 하네스가 말없이 뺀다는
            실측이 출처를 안 가린다. 안 더하면 이 유닛이 그 침묵을 팩 경로로
            되살린다
      - [x] 노드 · 워크스페이스 갈래는 U4 의 것을 안 고친다
- [x] 걸음 5 — `ensureType` 이 공용 자리 그대로다 (R22). 출처 셋이 같은 규칙을 받는다
- [x] 걸음 6 — 요청 안 한 팩 이름을 Note 로 남긴다 (R18).
      워크스페이스 Note 는 U4 의 것 그대로
- [x] 거절로 끝나는 모든 자리가 그때까지의 `Components` 를 함께 돌려준다 (U4 의 R11)

### Step 4 — `internal/enode/runner.go` — 가장자리와 기록

- [x] `openPack(j Job) packInput` 를 짓는다. **`readCannot` 옆이다** — 이 파일에
      이미 있는 유일한 파일 읽기 헬퍼가 그것이고, 팩도 같은 종류다
- [x] `j.Params.Pack` 이 비면 빈 `packInput` 이다 (R1). 파일을 안 연다
- [x] `filepath.Join(j.IO.In, name)` 을 연다. 없으면 `Err` 에
      `pack blob %s was not produced by this run` (R2).
      **이름을 `filepath.Base` 로 좁힌다** — blob 이름은 평평하지만 여는 쪽이
      그것을 가정하지 않는다
- [x] `defer f.Close()` 뒤 `readPack` 에 넘긴다. 상한은 `defaultPackLimits`
- [x] ①.5 — `runHarness` 의 ② 앞에서 `openPack` 을 부른다. 함수 머리 주석의
      아홉 걸음을 열로 고친다
- [x] ② 의 호출이 `resolveComponents(j, pk)` 가 된다. Notes 를 찍는 자리와
      오류 검사의 순서는 **안 바뀐다**
- [x] ⑧ — `ctx.Err()` 갈래 **뒤**에서 두 필드를 채운다 (R30).
      타임아웃도 exec 은 지났으므로 값이 찬다
- [x] `res.MCP` 는 `c.Servers` 의 이름을 정렬한 것이다. 0 개면 안 채운다 (nil)
- [x] `res.Pack` 은 `c.Pack.SHA256` 이다. `c.Pack` 이 nil 이면 빈 값

### Step 5 — `internal/enode/harness.go`

- [x] `HarnessResult` 에 `MCP []string` · `Pack string` 을 더한다.
      **둘 다 `omitempty`** — 팩도 서버도 없는 단계의 봉인이 오늘과 한 글자도 안 달라진다
- [x] 주석이 「요청한 것이 아니라 실린 것이다」를 적는다 (`ADR-005` 성질 4)
- [x] `ParseClaude` 와 `claudeEnvelope` 는 **안 건드린다** — 어댑터가 정하는
      값이 아니라 우리가 정해서 준 사실이다

### Step 6 — `internal/enode/claude.go` — 펴기

- [x] `const packDirName = "pack"`. **디렉터리 이름이 곧 스킬 이름의 접두다** —
      `pack:hello` 가 그 값에서 나온다는 것을 주석이 적는다
- [x] `writePack(dir string, p *Pack) error` — `<dir>/pack` 을 0700 으로 만들고
      `Files` 를 이름 그대로 0600 으로 쓴다. 파일마다 `MkdirAll(filepath.Dir)` (R24)
- [x] 검증을 다시 안 한다. `Pack` 은 `readPack` 만이 짓는다는 것을 주석이 적는다
- [x] `Instrument` 의 ③ 자리 — `c.Pack` 이 nil 이면 아무것도 안 한다 (R23).
      **`--plugin-dir` 도 안 붙는다** — 팩 없는 단계의 argv 가 오늘과 한 글자도
      안 달라야 CA1 의 서명이 산다 (R26 의 짝)
- [x] 실패는 치명이다 — `cannot extract the pack: %w`. `errAux` 로 안 감싼다 (R25)
- [x] 플래그는 **등호 형태** `--plugin-dir=<경로>` 다 (4절 ③)
- [x] 자리는 훅 설정 뒤 · 허용목록 앞이다 (R27). **조기 반환하지 않는다는
      U1 의 불변식이 그대로 산다** — 훅이 실패해도 팩과 허용목록을 마저 쓴다
- [x] `Instrument` 머리 주석의 「`<dir>/home/skills · agents` — U5 가 짓는다」를
      **홈 밖의 `<dir>/pack/`** 으로 고친다 (7절)

### Step 7 — `cmd/iapadapter/config.go`

- [x] `ExecutorConfig.Pack *PackConfig` (`yaml:"pack,omitempty"`)
- [x] `PackConfig{Fetch []string, Name string}`
- [x] `LoadConfig` — `Pack` 이 있는데 `Fetch` 가 비면 거절한다
      `executor.pack.fetch is required when executor.pack is set`.
      **설정 오류이지 「팩 없음」이 아니다**
- [x] `Name` 이 비면 `"pack"` 이다
- [x] `Name` 의 글자는 안 검증한다 (4절 ⑤)

### Step 8 — `cmd/iapadapter/contract.go`

- [x] `cfg.Executor.Pack` 이 nil 이면 `steps` 가 오늘 그대로다 (R31).
      `adapter_test.go` 의 `steps[0]`=plan · `steps[1]`=gate 가 안 빨개진다
- [x] 있으면 맨 앞에 명령 단계 하나 (R32) —
      `id: "pack"` · `uses: <executor.as>` · `run: <fetch>` · `out: [<name>]`
- [x] `plan` 에 `needs` 를 **안 적는다** (R33). `NeedsOf` 의 기본값이 직전
      단계라 계획이 팩 단계를 기다린다. 주석이 그 근거를 `contract.go:764-792`
      로 적는다
- [x] `success_when` 을 안 바꾼다 (R35)
- [x] `planPrompt` 에 blob 이름과 두 키를 알려주는 줄을 더한다 (R34).
      **영어다** — 프롬프트는 밖으로 나간다 (`CONVENTIONS.md` 2.1).
      글자는 4절 ④ 가 못 박는다
- [x] `requires` 는 안 는다 — 팩 단계가 `executor.as` 를 그대로 쓴다

### Step 9 — 실측 하나 (우리 코드가 지은 디렉터리를 하네스가 읽나)

- [x] 시험 안에서 tar 를 지어 `readPack` 에 넣고 `writePack` 으로 펴서
      **우리 코드가 지은 디렉터리**를 만든다
- [x] 그 디렉터리를 `claude --setting-sources "" --plugin-dir=<경로> ...` 에 물려
      `init` 줄의 `skills` 와 `slash_commands` 를 읽는다. **함대 없이 실물
      하네스로 잰다** (U4 의 Step 6 과 같은 꼴)
- [x] 재는 것 셋 — 이름이 `pack:hello` 인가 · 등호 형태가 서는가 ·
      `--strict-mcp-config` 를 뒤에 두어도 안 삼켜지는가 (4절 ③)
- [x] 결과를 `code-summary.md` 와 `decisions.md` ㉔ 에 적는다. **안 나오면**
      그 자리에서 멈추고 사용자에게 묻는다 — 답 1 의 근거가 무너진 것이다

### Step 10 — 시험

- [x] 새 파일 `internal/enode/pack_test.go` — `readPack` 의 표.
      악성 tar 를 **메모리에서** 짓는다. 디스크가 필요 없다
  - [x] R3 — gzip 을 투명하게 푼다 · tar 가 아닌 것은 거절 · gzip 안이 tar 가
        아니어도 거절
  - [x] R4 — 심볼릭 링크 · 하드 링크 · 장치 · FIFO 넷이 각자의 문구로 거절된다
  - [x] R5 · R6 · R7 — 절대경로 · `..` · 백슬래시 · 제어문자 · 드라이브 문자 · 빈 이름
  - [x] R8 — 받은 바이트가 넘는 경우와 푼 바이트가 넘는 경우 **둘 다**.
        gzip 폭탄이 압축 크기로는 안 걸리고 푼 크기로 걸린다
  - [x] R9 — 개수 상한. 디렉터리와 규약 밖 항목이 함께 세어진다
  - [x] R11 — `skills/` 와 `agents/` 의 곁 파일까지 실린다
  - [x] R12 — 깨진 `mcp.json` 은 거절 · `mcpServers` 없는 파일은 서버 0
  - [x] R13 · R14 · R15 — Note 셋의 문구. `settings.json` 의 문구가 따로다
  - [x] R16 — 실을 것이 0 인 팩의 거절 문구가 팩 이름을 담는다
  - [x] 2.2 — 해시가 **파일 전체**의 값이다. tar 꼬리 0 블록 뒤에 바이트를
        더 붙여 걸음 4 가 있을 때와 없을 때가 갈리는 것을 잰다
  - [x] R10 — `t.TempDir()` 를 주고 돌린 뒤 그 디렉터리가 비어 있다
- [x] `internal/enode/resolve_test.go` 에 더한다 (U4 의 파일)
  - [x] 팩의 서버가 `agent.mcp` 필터를 탄다 (R17) · 요청 안 한 이름은 Note (R18)
  - [x] 요청이 0 이어도 `c.Pack` 이 산다 (걸음 1 이 2 앞)
  - [x] `p.Err` 가 있으면 요청이 0 이어도 죽는다 (걸음 0 이 2 앞)
  - [x] R19 의 거절 문구 · R20 (요청 안 한 이름의 겹침은 안 죽인다)
  - [x] R21 — 팩이 워크스페이스를 이긴다. Note 가 남는다
  - [x] R22 — 팩 항목도 종류가 채워진다. `Pack.MCP` 의 원본이 안 바뀐다
  - [x] **기존 호출 전부를 두 인자로 고친다** — 인자가 늘어 확정 빨강이다
- [x] `internal/enode/instrument_test.go` 에 더한다 (U1 의 파일)
  - [x] 펴진 파일의 내용과 권한 0600 · 디렉터리 0700
  - [x] 팩이 없으면 `--plugin-dir` 이 argv 에 **없다**
  - [x] 훅 쓰기를 실패시켜도 팩이 펴진다 (U1 의 불변식)
  - [x] 펴기 실패가 치명으로 단계를 죽인다
- [x] 기록 두 필드 — 가짜 하네스로 도는 배선 시험.
      실린 이름이 정렬되어 있고, 팩 없는 단계의 봉인 JSON 에 두 키가 **아예 없다**
- [x] `cmd/iapadapter/config_test.go` — `pack` 이 비면 nil · `fetch` 가 비면 거절 ·
      `name` 이 비면 `pack`
- [x] `cmd/iapadapter/adapter_test.go` — 설정이 비면 오늘 그대로 · 팩이 있으면
      `steps[0]` 이 팩 단계이고 `out` 이 그 이름이며 `plan` 에 `needs` 가 없다
- [x] 배선 — `openPack` 이 없는 blob 을 만났을 때 **하네스가 안 뜬다**.
      스텁이 파일을 떨구게 해 「안 떴다」를 파일 없음으로 잰다

### Step 11 — 변이 일곱

넣고 **빨개지는지** 본다. 안 빨개지면 그 규칙을 재는 시험이 없는 것이다.

- [x] ① 걸음 4 의 드레인(`io.Copy(io.Discard, br)`)을 걷는다
- [x] ② R4 의 종류 검사를 걷고 이름 검사만 남긴다
- [x] ③ 푼 바이트 상한을 걷고 받은 바이트만 잰다
- [x] ④ R16 의 거절을 걷고 빈 팩을 통과시킨다
- [x] ⑤ 걸음 0 을 걸음 2 뒤로 옮긴다 (깨진 팩이 요청 0 에서 안 죽는다)
- [x] ⑥ 걸음 1 을 걸음 2 뒤로 옮긴다 (요청 0 에서 스킬이 안 펴진다)
- [x] ⑦ R17 의 필터를 걷고 팩의 서버를 전부 싣는다

### Step 12 — 게이트 CA0

- [x] `eval "$(scripts/testdb.sh)"` 뒤 `go test ./...` 전부 초록
- [x] 커버리지 — 패키지별 80% 미달 0. `internal/enode` 를 특히 본다
      (U4 뒤 **84.5%** 이고 이 유닛이 새 코드가 가장 많다)
- [x] `go vet ./...` · `gofmt -l` 이 빈다 · `go run ./scripts/glyphscan.go`
- [x] U+2605 전수 grep 0 · 크로스 빌드 셋 · 심볼 상한
- [x] 라우트 26 — `grep -rho 'mux\.HandleFunc\|mux\.Handle(' internal/api/*.go | wc -l`
- [x] 워킹트리 청결. 커버리지 실행이 바꾸는 `cmd/enodectl/probe.lock` 은 되돌린다

### Step 13 — 문서와 팩

- [x] `decisions.md` 6절에 실측 행 ㉔ (FD 계획 1.1 ~ 1.5 와 Step 9)
- [x] `decisions.md` 2절 워크스페이스 `.claude/skills` 행을 값으로
- [x] `features.md` 5절의 미정 하나를 걷는다 · 3.6 의 펴는 자리를 고친다
- [x] `unit-of-work.md` 5절의 `claude.go` 줄
- [x] `scene-gates.md` 3절 CA5 의 명령 넷
- [x] `component-methods.md` 2.1 의 `Pack.MCP` 타입과 `PackFile.Mode`
- [x] 파일 행렬 — 2.2 에 ①.5 · U5 열에 새 시험 파일 행
- [x] `construction/pack/code/code-summary.md`
- [x] `aidlc-docs/taeels/aidlc-state.md` 의 U5 절과 `audit.md`
- [x] 이 계획의 체크박스를 전부 `[x]` 로

---

## 3. 갈래를 안 나눈다

`resolveComponents` 의 **인자가 하나 는다.** 그 순간 `runner.go` 의 호출과
`resolve_test.go` 의 호출 전부가 같은 시점에 컴파일된다. `Pack` 의 필드가
없으면 `readPack` 이 안 서고, `HarnessResult` 의 필드가 없으면 ⑧ 이 안 선다.
갈래마다 컴파일되는 시점이 하나뿐이라 나누는 비용이 이득보다 크다 —
U1 ~ U4 와 같은 판정이다.

---

## 4. 이 계획이 정하는 것 — FD 에 없던 자리 다섯

```text
   ①  readPack 이 팩 이름을 인자로 받는다
      R3 과 R16 의 문구가 팩 이름을 담는데, 바깥에서 감싸면 게이트가 찾는
      글자가 그대로 안 나온다 (pack <이름> is not a tar archive).  io.Reader
      하나로 악성 tar 를 메모리에서 짓는다는 시험의 값은 안 바뀐다

   ②  R4 의 종류 이름을 글자로 못 박는다
      symlink · hard link · device · fifo 넷과 모르는 종류의 tar type '<글자>'.
      문구가 pack entry %s is a %s; only regular files are read 이므로 이 글자가
      곧 사람이 읽는 사유다.  모르는 종류의 낱말을 계획은 unknown entry type 으로
      적었고 코드는 tar 의 글자를 그대로 보이는 쪽으로 갔다 — is a 뒤에 놓으면
      앞이 영어가 안 되고, 이름을 지어내면 그 이름으로 찾을 문서가 없다

   ③  --plugin-dir 은 등호 형태다
      --mcp-config 와 같은 이유다 (ADR-034 §3 ② 의 가변인자 함정).  이 자리는
      Instrument 의 ③ 이라 뒤에 --strict-mcp-config 가 따라붙는데, 띄어 쓴
      형태가 가변인자면 그것을 삼킨다.  실측(1.2)은 공백 형태로 쟀으므로
      Step 9 가 등호 형태를 다시 잰다 — 안 서면 거기서 멈춘다

   ④  planPrompt 가 더하는 글자
      두 줄이고 영어다.  blob 이름을 이름으로 알려주고 두 키를 나열한다:
        A pack tar is produced by the first step as blob "<name>".
        Any agent step that should use it must set agent.pack: "<name>"
        and in.from: ["<name>"].
      이름을 설정에서 끌어오므로 문장이 설정과 안 갈린다

   ⑤  PackConfig.Name 의 글자를 LoadConfig 가 안 검증한다
      blob 이름의 정본은 contract.Validate 이고 그것이 제출에서 거절한다.
      여기서 또 세면 규칙이 두 벌이 된다 (CONVENTIONS 1.4 의 그 규율)
```

---

## 5. 이 단계가 안 하는 것

```text
   claim.go            행렬 밖이다.  팩은 runHarness 가 연다 (답 3=A)
   Decode              짝 팩의 것이다 (constraints.md 접점 절)
   계약 어휘의 추가      0.  U2 가 닫았다 — agent.pack 하나이고 안 는다 (6절 ⑫)
   새 라우트 · 새 전송   0.  팩은 이미 있는 blob 경로로만 흐른다
   runctl submit --pack 이월 (features.md 4절)
   기대 다이제스트 검증  이월 (decisions.md 6절 ③)
   팩 캐시 · system.md  이월
   새 Reason           안 만든다.  거절은 ReasonError 로 간다
   보조 등급           안 는다.  errAux 로 감싸는 자리는 훅 하나뿐이다 (⑪)
```

---

## 6. CA5 와 CA6 은 사람이 잰다

Step 9 가 재는 것은 **우리 코드가 지은 디렉터리를 하네스가 읽나**이고, CA5 는
**노드와 Mediator 를 지나 그 팩이 실제로 실리나**다. CA6 은 사내 함대에서
`runctl record` 로 푼 `steps/NN-*.json` 의 두 필드를 읽는다.

**집행자는 이 유닛을 구현하지 않은 사람이다** (`scene-gates.md` 2절 머리).
**눈 검증을 보류로 안 넘긴다** (같은 문서 4절) — 앞 팩의 CP6 이 그렇게 닫혔고
그 결함이 사내 실측에서야 드러났다.
