# enode 기능 추가 요구사항 정의서 — 2판

1판에서 **미정**이던 값을 `decisions.md` 가 닫았다. 이 문서는 값을 다시 적지
않고 그 표를 가리킨다. **미정으로 남은 것은 4절의 이월뿐이다.**

```text
   decisions.md         이미 정해진 것.  다시 논의하지 않는다
   unit-boundaries.md   권장 유닛 경계 + 만지는 파일
   scene-gates.md       장면 조각 게이트 CP0 ~ CP4.  이 문서의 수용 기준이다
   canon.md             enode-design 과의 연결.  어긋나면 INVARIANTS 가 이긴다
   constraints.md       안 만드는 것 여덟 범주
   design/              화면 여덟 장과 그리는 규칙
```

## 1. 개요

### 1.1 무엇을 만드나

enode 는 **부재중인 기능 담당자의 대리인**이다. 복제할 수 없는 자원(개발보드,
개발자가 쌓아온 컨텍스트)에 추론이 찾아가게 한다. 지금은 그 함대가 **보이지
않고, 소유자가 통제할 수 없고, 자원이 없으면 그냥 실패한다.**

이번에 그 셋을 메운다. 넷을 만들고 그 넷이 장면 하나를 완주한다.

### 1.2 구성 요소 (기존)

```text
   Mediator    매칭 · 임대 · 시퀀싱 · Record 봉인.  PostgreSQL.  Listen :8080
   enode       노드 데몬.  광고 · claim 롱폴 · 하네스 실행.  밖으로 듣지 않는다
   runctl      제출 · 상태 · Record · 취소 · shape.  무상태
   enodectl    노드 로컬 제어.  setup · list · id · start · stop · logs · status
```

### 1.3 이번에 더하는 것

```text
   3.1 관측    중앙 현황판 · 호스트 제어판
   3.2 제어    대기열 · 소유자 자원 제어권 (drain)
   3.3 접수    Mediator MCP
```

실행(역할 주입)만 **이월**이다 — 4절.

**3.3 은 2판을 판 뒤에 들어왔다.** 이월이던 것을 진행자가 되돌린 것이고
근거는 `decisions.md` 5절에 있다. 요약하면 하나다 — 이 축이 **아무것도 안
딛는다.** `ADR-022` §9.3 이 `S1 MCP 어댑터`를 「독립」으로 적고 「이미 있는
REST 를 도구로 감싼다 — 새 의미가 0 개다」로 못 박았다. 이월의 이유였던
「전송 · 토큰 전달 미정」은 값의 문제였고 5절이 그 값을 닫았다.

### 1.4 정본

`canon.md`. 설계 정본은 서브모듈 `enode-design/` 이고 어긋나면
`protocol/INVARIANTS.md` 가 이긴다.

### 1.5 가치의 고정점

`scene-gates.md` §1 의 `dhseo` 장면이다. **완결성은 그 장면이 끝까지 도는가로
판단한다.** 장면의 일은 보드의 **LED 점멸 패턴을 바꾸는 ko 또는 프로그램을
작성**하고 그 보드에서 돌려 확인하는 것이다 — 에이전트가 한 일이 눈에 보이는
물건으로 남는다.

---

## 2. 용어 정의

- **노드** — 광고를 보내는 기계 하나. `node_id` 로 식별한다
- **임대** — Mediator 가 노드를 한 Run 에 배타 점유시킨 것. `leases.node_id` 가
  기본키라 **노드 하나에 임대 하나**다
- **소유자** — 그 노드를 함대에 내놓은 사람. `nodes.principal` 에 있다. 식별이지
  인증이 아니다 (`ADR-015` §1)
- **광고** — 노드가 스스로 「나는 무엇을 할 수 있다」를 보내는 것. 매번 전부이고
  **만료된다**
- **정책** — 소유자가 「이 노드를 지금 빌려주나」를 정한 것. 노드의 파일이 정본이고
  **만료되지 않는다**. 첫 입주자는 `drain`
- **drain** — 소유자가 자기 노드를 반납받는 것. 새 임대를 막는다. 단계 도중에 끊지
  않는다 — `graceful` 은 도는 Run 을 끝까지 두고, `at-boundary` 는 경계에서 닫는다
- **단계 경계** — 한 단계가 끝나고 다음이 시작되기 전. `RUNNING -> RUNNING` 이
  비멱등이라 **중단이 안전한 유일한 지점**이다
- **QUEUED** — 후보는 있는데 전부 점유돼서 기다리는 Run 의 상태. 아무것도 쥐지 않는다

---

## 3. 핵심 기능 요구사항

### 3.1 관측

#### 3.1.1 중앙 현황판

**목적**: 함대 전체가 지금 무엇을 하고 있는지 한 화면에서 본다.

**기능 요구사항**:

- **함대 목록** — `GET /v1/nodes` **신규**
  - 응답 모양은 `ADR-065` 결정 그대로 — `observed_at` · `nodes[]` 의 `node_id` ·
    `label` · `instance` · `capabilities` · `seen_at` · `expires_at` · `lease{run_id, not_after}`
  - 여기에 `draining` 을 더한다 (`decisions.md` 2절 · `canon.md` §3 선행 조건)
  - `principal` 을 내지 않는다 · 필터를 주지 않는다 · 만료되지 않은 광고만 낸다
- **Run 목록** — `GET /v1/runs` **신규**
  - 응답 모양과 필터는 `decisions.md` 2절. 시간 역순. `limit` 기본 100. 페이지네이션 없음
  - 대기열(3.2.1)을 보이려면 필수다
- **작업 그래프** — Run 하나의 단계 의존 그래프
  - 오늘 이미 나온다. `GET /v1/runs/{id}` 의 `steps[]` 와 `needs[]` 가 간선이다
  - `StepView` 에 `chosen` 을 더한다 — DB 엔 있고 뷰에 없다 (`ADR-060`)
- **임대 표시** — `GET /v1/nodes` 의 `lease` 에 내장. 별도 표면 없음

**UI/UX 요구사항**: `design/README.md` 의 규칙을 따른다.
- 화면 S0 · S0b · S1 · S1b · S2. 함대 카드 격자 + Run 목록이 한 화면(S1)
- 카드에 — `label` · 살아있나 · 임대 유무 · 지금 무슨 단계 · 언제 풀리나 · draining 배지
- 중앙은 **읽기 전용**이다. drain 을 걸 수 없다
- 폴링. 간격과 임계값은 `decisions.md` 2절. 화면은 「마지막 갱신 시각」을 보인다

**데이터 관리**: 새 저장소 없음. 전부 기존 행을 내보내는 것이다

**보안 요구사항**: `GET /v1/runs` · `GET /v1/nodes` 는 Mediator 토큰으로 보호 —
지금 표면과 같은 규칙. 정적 파일(`GET /ui/`)은 Mediator 에 embed 하고 **무인증**
이다. 화면이 S0 에서 토큰을 받아 `Authorization` 헤더로 두 라우트를 부른다.
토큰은 브라우저 세션 저장소에만 둔다

**수용 기준**: `scene-gates.md` CP1. 노드가 여섯이면 카드가 여섯 뜬다. 임대된
노드는 언제 풀리는지가 보인다. 대기 중인 Run 이 목록에 뜬다. draining 노드는
「있는데 안 빌려준다」로 보인다

#### 3.1.2 호스트 제어판

**목적**: 노드 소유자가 자기 기계에서 enode 를 보고 통제한다.

**기능 요구사항**:

- **위치** — `enodectl serve <name>`. 별도 프로세스. 새 실행파일 없음 (`decisions.md` 1절).
  이름은 기존 하위 명령과 같은 규칙 — `names()` 가 하나면 생략, 여럿이면 필수
- **프로세스 제어** — `enodectl` 의 `status` · `start` · `stop` · `logs` 를 감싼다
- **신원 표시** — `node_id` · `label` · `principal` · `instance` 전부. 자기 기계다
- **탐지 능력 표시** — `Detect()` 결과를 **읽기 전용**으로 보인다. 편집 자리를
  만들지 않는다 — 광고는 매번 전부이고 다음 광고가 덮는다 (`ADR-017` 결정 3)
- **현재 작업** — 임대 유무 · 임대한 Run · `not_after` · `CLAIMED` 단계 이름 ·
  회차(`attempt`) · 시작 시각. 출처는 Mediator 조회 (`decisions.md` 2절)
- **drain 토글** — 3.2.2 를 부른다. 걸기 · 모드 고르기 · 풀기. 건 뒤에는 현재
  모드와 「풀기」가 보인다 (`design/README.md` §7)
- **Mediator 연결 상태** — 마지막 응답 시각. 끊기면 drain 통보가 늦어진다는 사실

**기존 코드에서 확인된 것**:
- `internal/enode` 에 `net.Listen` 이 **0건**이다. 노드는 밖으로 듣지 않고 Mediator
  로 나가기만 한다. **HTTP 서버를 처음부터 세워야 한다** — `enodectl serve` 가 그것이다
- `cmd/enodectl/main.go` 의 `names` · `pidOf` · `identityOf` · `ownsConfig` ·
  `signalStop` 과 `proc_unix.go` · `proc_windows.go` 의 `processAlive` 가 프로세스
  제어의 재료다. `internal/panel` 로 내려서 같이 쓴다

**보안 요구사항**:
- 바인딩 기본값 **`127.0.0.1:8081`**. LAN 노출은 `--listen` 명시로만. 켜면 정책
  파일의 `panel_token` 이 필수다 — 없으면 `serve` 가 뜨지 않는다. Mediator 토큰을
  재사용하지 않는다 (그건 함대 전체의 신뢰 경계다). `decisions.md` 2절
- 인증 없음이 기본인 이유 — 그 기계에 접속한 것이 소유의 증거다 (`ADR-063` §3)
- 정책 파일은 소유자만 쓸 수 있는 권한으로 만든다

**수용 기준**: `scene-gates.md` CP3 · CP4. 노드가 도는 상태에서 열면 임대 ·
현재 단계 · 회차가 보인다. 멈춘 상태에서 열면 그 사실이 보이고 `start` 로 띄울
수 있다. 기본 바인딩으로 띄우면 다른 기계에서 접속되지 않는다

### 3.2 제어

#### 3.2.1 대기열

**목적**: 자원이 없을 때 죽지 않고 기다린다.

**맥락**: `409`(후보는 있는데 전부 점유)가 지금 즉시 `FAILED` 다
(`internal/match/match.go` `CodeAllBusy`). Run 이 종료하고 Record 가 봉인되며
재시도가 호출자 몫이다. 보드가 하나뿐이면 두 번째 요청은 항상 죽는다.

**기능 요구사항**:

- **`QUEUED` 상태 추가**
  - 매칭에서 점유 실패(`409`) -> `QUEUED`. 제출 응답은 `202 Accepted`
  - 자원이 풀리면 매칭을 다시 하고, 되면 `RUNNING`
  - 코드는 `ALLOCATING` 을 저장하지 않는다 — `submit` 이 매칭 뒤 바로 `RUNNING` 을
    적는다(`api.go`). `INVARIANTS` §1.1 의 `RESOLVING` · `ALLOCATING` 은 그 한
    함수 안의 순간이다. `QUEUED` 는 그 자리에 실제로 저장되는 첫 중간 상태다
  - `422`(함대에 아예 없다)는 그대로 `FAILED`
- **`READY` 는 만들지 않는다** (`ADR-064` §1.1)
- **공정성은 FIFO 하나.** 도착 순서. 정책 틀을 만들지 않는다. 우선순위는 이월
- **대기 상한 없음.** 빠져나가는 길은 `cancel` 뿐
- **깨우기** — 임대가 지워지는 모든 지점 뒤에서 `store.WakeQueued` 를 부른다.
  `postResult` · `postCancel` · `Reap` · 재기동 감지 · 부분 반납 · drain 해제를 받은
  광고 처리 (`decisions.md` 1절). 주기에 얹지 않고 지점에 건다
- **매칭에서 draining 노드 제외** — `busy` 에 `DrainingNodes` 를 합친다

**데이터 관리**:
- `runs.state` 어휘가 하나 는다
- **크래시 복구** — `QUEUED` 는 임대도 노드도 쥐지 않으므로 재기동 후에도 `QUEUED`
  가 참이다. 기동 시 `WakeQueued` 를 한 번 부른다

**수용 기준**: `scene-gates.md` CP2. 노드 하나에 Run 둘을 던지면 둘째가 `202` ·
`QUEUED` 다. 첫째가 끝나면 둘째가 자동으로 `RUNNING` 으로 간다. 함대에 없는
능력을 요구하면 여전히 `422` · `FAILED` 다. 대기 중인 Run 이 `GET /v1/runs` 에
뜬다. `runctl status` 가 `QUEUED` 를 표시한다

근거 `ADR-064`

#### 3.2.2 소유자 자원 제어권 (drain)

**목적**: 노드 소유자가 자기 자원을 돌려받는다.

**맥락**: `nodes.principal` 에 소유자가 기록돼 있는데 그 열을 읽는 곳이 없다.
자원이 한 방향으로만 흐르고 소유자가 되찾을 길이 없다.

**기능 요구사항**:

- **정책 자리** — 광고와 별개로 소유자가 정하는 것. **정본은 노드의 파일**이다.
  위치와 형식은 기존 `enode` 설정 파일 옆, 같은 형식 — Functional Design 몫
- **경로** — 데몬이 광고 직전에 정책 파일을 읽어 광고에 싣는다. Mediator 는
  `UpsertAdvert` 에서 `nodes` 열에 복사본을 두고, 광고 응답에 되돌려 준다.
  데몬은 응답의 값을 `Worker` 에 넘긴다. **중앙 라우트 신설 없음**
- **두 모드** (`decisions.md` 1절)
  - `graceful` (기본) — 새 임대만 막고 현재 것은 자연 종료를 기다린다
  - `at-boundary` — 도는 단계는 끝까지 간다. 결과를 보고한 뒤 임대를 놓고 더
    집지 않는다. 놓인 Run 은 Mediator 가 `postResult` 끝에서 기존 취소 경로
    (`store.Cancel`, 사유 `drain:<node_id>`)로 닫는다 — 끝난 단계의 산출은
    Record 에 남고, 남은 단계는 새 Run 으로 다시 낸다 (`decisions.md` 1절)
- **해제** — 소유자가 명시적으로 푼다. 도는 Run 이 끝나도 자동으로 안 풀린다.
  풀면 다음 광고에서 후보로 돌아가고 `WakeQueued` 가 대기 Run 을 보낸다
- **매칭에서 제외** — draining 노드는 후보에서 빠진다. **광고는 계속 보낸다.**
  현황판에 「있는데 안 빌려준다」로 보여야 한다
- **지연 상한 없음** — 바닥(도는 단계의 남은 시간 + 광고 주기)이 곧 값. 더 급하면
  제어판에서 `stop` — 그쪽이 명시적 폐기다

**단계 도중에 끊지 않는다.** `graceful` 은 도는 Run 을 죽이지 않고, `at-boundary`
는 경계에서 닫되 끝난 단계의 산출을 남긴다. 둘 다 `I4` 를 안 건드리고 「재개하지
않는다」와도 안 부딪힌다. 그 점이 선점과 갈리는 자리다 — 선점은 진행분 폐기다.

**보안 요구사항**:
- 소유 증명은 **그 기계의 파일을 쓸 수 있다**는 것이다 (`ADR-042` · `ADR-063` §3)
- 중앙에서 소유권을 판정하지 않는다 (`constraints.md` §4). 토큰만 있으면 남의
  노드를 뺄 수 있는 표면을 만들지 않는다

**수용 기준**: `scene-gates.md` CP3. drain 을 걸면 새 Run 이 그 노드로 배정되지
않는다. `graceful` 로 걸면 도는 Run 이 끝까지 간다. `at-boundary` 로 걸면 다음
단계 경계에서 임대가 풀린다. 풀린 노드는 drain 을 풀기 전까지 후보가 아니다.
풀면 다시 후보가 되고 대기 Run 이 집는다

근거 `ADR-063`

---

### 3.3 접수

#### 3.3.1 Mediator MCP

**목적**: 사용자의 Claude 가 함대에 직접 붙는다. `runctl` 로 접수하던 것을
MCP 로도 접수하고, **현황판이 그리는 것과 같은 자료**를 도구로 낸다.

**맥락**: 이 제품의 진단이 「MCP 가 발산한다」였다. 그 진단을 해 놓고
**MCP 를 하나 만든다** — `N` 개를 없애려고 `1` 개를 만드는 것이다
(`prep/GOALS.md`). 2판을 팔 때는 이월이었고 2026-09-05 에 범위로 들어왔다
(`decisions.md` 5절).

**기능 요구사항**:

- **전송은 stdio 하나.** JSON-RPC 2.0 을 표준 입출력으로 나른다.
  streamable HTTP 를 안 연다 — 포트도 TLS 도 안 늘린다 (`constraints.md` §6)
- **자리는 `runctl mcp`.** 새 실행파일을 안 만든다. `packaging/` 이 안 바뀐다
- **토큰은 환경변수.** `ENODE_MEDIATOR` · `ENODE_TOKEN` 을 `runctl` 과 같은
  규칙으로 읽는다. **도구 인자로 안 받는다** — 받으면 토큰이 대화 기록과
  로그에 남는다
- **도구 여덟** — `capabilities.list` · `fleet.list` · `runs.list` ·
  `run.submit` · `run.plan` · `run.cancel` · `record.get` · `run.get`.
  전부 기존 REST 를 감싼다 (`decisions.md` 5.3)
- **새 의미가 0 개다** (`ADR-022` §9.3). 어휘를 안 만든다. 만드는 것은 통로다
- **새 Go 의존 0.** 프로토콜을 손으로 짠다.
  `internal/runctl/client.go` 를 그대로 재사용한다

**데이터 관리**: **없다.** 이 유닛은 DB 를 안 만진다. 스키마 변경 0 ·
새 라우트 0 · Mediator 무변경이다.

**안 만드는 것**:
- 트랜스크립트 구독. 툴은 한 방에 답하는 물건이라 구독 표현이 별도 결정이고
  여전히 이월이다
- streamable HTTP · 새 인증 경로 · 새 REST 라우트

**수용 기준**: `scene-gates.md` CP5. `runctl mcp` 가 stdio 로 서고
`tools/list` 가 여덟을 낸다. `run.submit` 으로 낸 Run 이 현황판 S1 에 그대로
뜬다. `fleet.list` 가 `GET /v1/nodes` 와 글자까지 같은 자료를 낸다.
`go.mod` 와 `packaging/` 이 안 바뀐다

---

## 4. 범위

### 필수 — 다섯

```text
   3.1.1 중앙 현황판     GET /v1/runs · GET /v1/nodes (+draining · lease) · chosen · 화면
   3.2.1 대기열          409 -> QUEUED · 202 · FIFO · 깨우기
   3.2.2 drain           정책 파일 + 광고 경로 + 두 모드 + 해제
   3.1.2 호스트 제어판    enodectl serve + 프로세스 제어 + 현재 작업 + drain 토글
   3.3.1 Mediator MCP    runctl mcp — stdio 로 도구 여덟.  기존 REST 를 감싼다
```

**앞의 넷이 `dhseo` 장면을 한 장면으로 만든다** — `scene-gates.md`.

**3.3.1 은 그 장면 밖이다.** 장면에 안 나오므로 `CP4` 의 조건이 아니고 자기
게이트 `CP5` 를 따로 갖는다. 딛는 것이 3.1.1 의 라우트뿐이라 **CP1 이 초록인
지금 이미 착수 가능**하고, U-drain · U-제어판과 파일이 하나도 안 겹쳐
병렬로 돈다.

### 이월 — 이번에 만들지 않는다

```text
   트랜스크립트           runner.go 의 stdout tee.  S4 화면은 자리만
   역할 주입              계약 문법이 늘어난다.  --agents 실측은 prep/GOALS.md 에 있다
   우선순위               FIFO 만.  큐 순서에만 닿는다는 경계는 ADR-064 §2.2
```

1판의 3.3 · 3.4 절은 `prep/GOALS.md` 에 남겨 두었다. 요구사항이 아니라 다음
회차의 후보다.

---

## 5. 미정

**없다.** 1판의 미정 16개는 `decisions.md` 1절과 2절이 닫았고, 이월된 것은
4절과 `decisions.md` 4절에 있다. 게이트에서 새 미정이 생기면 그것은 **계약
(서명 · 파일 · 응답 모양)에 걸리는 것**이어야 하고, 진행자가 `decisions.md`
에 행을 더한다.
