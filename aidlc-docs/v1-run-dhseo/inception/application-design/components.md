# 컴포넌트 — 정의와 책임

AI-DLC Application Design 산출물이다. **정본은 `requirements/` 팩**이고 이
문서는 이번 회차가 만드는 컴포넌트의 겉면과 책임을 적는다. 값은 팩을 가리키고
메서드 시그니처는 `component-methods.md` 가 진다.

결정 근거 — `application-design-plan.md` 4절 Q1~Q5 (전부 A).

---

## 1. 컴포넌트 지도

```text
   새 패키지 (이번 회차가 만든다)
     internal/panel     호스트 제어판 HTTP 서버
     internal/api/ui    /ui/ 아래 정적 화면 (현황판 · 데모 표면)
     internal/mcp       runctl mcp — stdio JSON-RPC 어댑터
     internal/proc      플랫폼 프로세스 제어 원장 (Q1 = A)

   새 파일 (기존 패키지 안)
     internal/api/nodes.go   GET /v1/nodes 핸들러
     internal/api/runs.go    GET /v1/runs 목록 핸들러
     internal/api/demo.go    데모 제출 핸들러 (allow-list · 토큰 주입) (Q5 = A)

   만지는 기존 패키지
     internal/store     QUEUED 어휘 · WakeQueued · submitter · draining 재료
     internal/match     draining 제외 (busy 합침, if !dry 안)
     internal/enode     drain 정책 파일 · 광고 경로 · 링 파일 tee/읽기
     internal/runctl    Nodes · Runs 클라이언트 메서드 (Q2 = A)
     cmd/enodectl       serve 하위명령 (위임 exec)
     cmd/enode          panel 하위명령 (serve 가 exec 하는 대상)
     cmd/mediator       /ui/ embed 마운트 · 새 라우트 등록
```

---

## 2. 새 패키지

### 2.1 internal/panel — 호스트 제어판 서버

**목적**: 노드 소유자가 자기 기계에서 enode 를 보고 통제한다 (`3.1.2`).

**책임**:

- HTTP 서버를 `127.0.0.1:8081` 기본으로 띄운다. LAN 은 `--listen` 명시로만이고
  그때 `panel_token` 이 필수다 (`decisions §2`)
- 신원 표시 (`node_id` · `label` · `principal` · `instance`) · 탐지 능력 표시
  (읽기 전용 · 잰 시각 함께 · 출처는 6절 열린 미정)
- 현재 작업 — Mediator 조회로 채운다 (`internal/runctl.Client`)
- drain 토글 — 정책 파일을 쓴다 (걸기 · 모드 · 풀기)
- 트랜스크립트 카드 — 노드 링 파일을 1초마다 읽는다. 지난 작업은
  `GET /v1/runs` 를 받아 자기 `node_id` 로 거르고 봉인 tar 를 푼다
- 프로세스 제어 — `status` · `start` · `stop` · `logs`. `stop` 은 도는 Run 을
  먼저 `cancel` 한다 (`decisions §7.2`)

**인터페이스**: `Config` · `Server` · `New` · `Handler`. 정적 자산은 없다 —
제어판 화면 자산은 이 패키지가 embed 한다 (현황판 자산 `internal/api/ui` 와 별개).

**임포트 경계**: `internal/store` · `internal/api` 를 임포트하지 않는다. Mediator
는 `internal/runctl.Client` 로 HTTP 로만 본다. 프로세스 제어는 `internal/proc`
를 쓴다.

### 2.2 internal/api/ui — 중앙 현황판·데모 정적 화면

**목적**: 실제 현황판(읽기 전용 · `3.1.1`)과 대회 데모 표면(`3.4`)의 정적
자산. 이 패키지는 정적 파일만 낸다 — 쓰기 경로가 없다.

**책임**:

- `/ui/` 아래 정적 파일을 embed 로 낸다. Mediator 가 마운트한다 (`decisions §2`)
- 화면 자산 — S0~S2 현황판 · 데모 대시보드(게스트·온보딩·가이드투어) ·
  3D 뷰(S6·S7 · shonsin pen.dev) · 웹캠 플로팅 임베드
- 브라우저가 S0 에서 토큰을 받아 `GET /v1/nodes` · `GET /v1/runs` 를 부른다.
  데모 게스트는 토큰 입력 대신 서버측 주입이라 브라우저에 실 토큰이 없다
- 읽기 전용은 **실제 현황판의 UX 태세**다 — 브라우저에서 submit·drain 을 못
  한다 (`3.1.1`). 데모 대시보드는 「새 작업」 쓰기를 갖되 그 경로는 이 패키지가
  아니라 `internal/api/demo.go` 다 (서버측 allow-list · 토큰 주입 · Q5).
  **데모 Mediator 자체는 읽기 전용이 아니다** — 고정 시나리오 제출을 받아
  rpi·mac 에서 돌리는 일회용 인스턴스다 (`decisions §8.3` · `constraints`
  「공개 쓰기는 데모 표면에만 산다」)

**인터페이스**: `//go:embed assets` + `Handler() http.Handler`. 서버 로직이 없다.

**임포트 경계**: `internal/store` 를 임포트하지 않는다 — 화면은 정적 파일이다.
데모 제출의 서버 로직은 여기 살지 않는다 (`internal/api/demo.go` — 4절).

### 2.3 internal/mcp — Mediator MCP 어댑터

**목적**: 사용자의 Claude 가 stdio 로 함대에 붙는다. 기존 REST 를 도구로
감싼다 (`3.3.1` · `ADR-022 §9.3` — 새 의미 0).

**책임**:

- `runctl mcp` 가 이 패키지를 stdio JSON-RPC 2.0 루프로 띄운다 (`decisions §5.2`)
- 도구 열 개를 `tools/list` · `tools/call` 로 낸다. 전부 기존 REST 를 감싼다
- 토큰은 환경변수 (`ENODE_MEDIATOR` · `ENODE_TOKEN`) — 도구 인자로 안 받는다
- `fleet.list` · `runs.list` 는 Mediator 응답 본문을 그대로 흘려보낸다 (Q3 = A —
  글자까지 같다 · CP5)

**인터페이스**: `Server` · `New(client)` · `Serve(ctx, in, out)` + 도구
디스패치 표.

**임포트 경계**: `internal/runctl.Client` 를 재사용한다 (Q2 = A). 새 HTTP
클라이언트도 새 Go 의존도 안 짓는다.

### 2.4 internal/proc — 플랫폼 프로세스 제어 원장 (Q1 = A)

**목적**: 데몬 프로세스를 보고 멈추는 플랫폼 원장을 제어판과 CLI 가 **같이
쓴다** (`3.1.2`).

**책임**:

- 빌드 태그 짝(`!windows` / `windows`)으로 `processAlive` · `signalStop` ·
  `ownsConfig` 를 낸다 — 오늘 `cmd/enodectl/proc_unix.go` · `proc_windows.go` 에
  있는 것을 내린다
- `net/http` 를 임포트하지 않는다 — `cmd/enodectl` 이 이 패키지를 임포트해도
  `enodectl.exe` 의 심볼 상한(net/http ≤50)이 안 깨진다

**인터페이스**: 프로세스 원장 함수들 (플랫폼 짝). HTTP 서버가 없다.

**임포트 경계**: `cmd/enodectl` 과 `internal/panel` 이 둘 다 임포트한다.
`internal/panel` 도 `net/http` 를 여기서 안 끌어온다 (제어판의 net/http 는
패널 자신의 것이다).

**이름 주의**: `internal/proc` 는 제안 이름이다. Units Generation 이 파일 행렬을
낼 때 최종 이름을 굳힌다.

---

## 3. 만지는 기존 패키지

### 3.1 internal/store

- `QUEUED` 를 실제 저장 상태로 추가한다 (`ADR-064` — 오늘 죽은 상수도 아니고
  아예 없다). `runs.state` 어휘가 넷에서 다섯이 된다
- `WakeQueued` 를 새로 낸다 — 임대가 지워지는 여섯 지점 뒤에서 부른다 (`decisions §1`)
- QUEUED 행 생성 경로 — submit 두 자리(매처 거절 · `ErrNodeTaken` 롤백)가
  이것을 부른다. 뒤 자리는 오늘 행조차 안 만든다
- `submitter` 표시 필드를 `runs` 에 더한다 (additive · `decisions §8.2 Q3`)
- 관측 스냅숏 — `GET /v1/nodes` · `GET /v1/runs` 목록의 재료. draining 제외 집합
- `StepView.chosen` 을 select 한다 (`ADR-060` — DB 엔 있고 뷰에 없다)
- getRun 의 `requires` · `as` (`ADR-069` — QUEUED 가 무슨 능력을 기다리나)

**크래시 복구**: `QUEUED` 는 임대·노드를 안 쥐므로 재기동 후에도 참이다. 기동
시 `WakeQueued` 를 한 번 부른다 (`decisions §2` · `ADR-064 §6`).

### 3.2 internal/match

- draining 노드를 후보에서 뺀다. 순수 매처는 그대로 두고, api submit 이
  `busy` 에 draining 집합을 합친다 — **`if !dry` 안**이다 (`ADR-063 §6` ·
  `api.go:389~395`). dry-run 은 draining 을 안 본다

### 3.3 internal/enode

- drain 정책 파일을 광고 직전에 읽어 광고에 싣는다 (`decisions §1`). 정본은
  노드의 파일. 위치·형식은 FD
- 하네스 stdout/stderr 를 고정 크기 링 파일에 tee 한다 — 에이전트 단계
  (`runner.go`)와 명령 단계(`claim.go`) 둘 다 (`decisions §6.2·§6.3`)
- 탐지 능력 값은 `Capabilities{Caps, At}` 에 데몬 메모리로만 있다 (`ADR-068`).
  제어판이 이것을 읽는 계약은 열린 미정 (6절)

### 3.4 internal/runctl (Q2 = A)

- `Client` 에 `Nodes` · `Runs` 두 메서드를 더한다 — 이 회차가 만드는 새 라우트다.
  `panel` 과 `mcp` 가 같은 클라이언트를 쥔다. 응답 본문을 그대로 돌려주는 형태라
  MCP 의 글자 일치(Q3)를 이 자리에서 보장한다

### 3.5 cmd/enodectl · cmd/enode

- `cmd/enodectl` 에 `serve <name>` 하위명령을 더한다 — HTTP 서버를 직접 안
  세우고 형제 `enode` 의 제어판 프로세스를 exec 위임하고 표준 입출력을 잇고
  종료 코드를 그대로 쓴다 (`decisions §1` · `constraints` — 심볼 상한).
  `cmd/enodectl/setup.go` 가 같은 모양을 보인다
- `cmd/enode` 에 제어판 하위명령을 더한다 — `internal/panel` 을 임포트해 서버를
  띄운다. `cmd/enode -> internal/panel` 은 허용이다 (`internal/enode -> internal/panel`
  만 금지 · `constraints`)

### 3.6 cmd/mediator

- `/ui/` 아래에 `internal/api/ui` 의 embed FS 를 마운트한다 (`decisions §2`)
- 새 라우트를 등록한다 — `api.go` 에는 등록 줄만, 핸들러는 새 파일 (`constraints`)

---

## 4. 새 핸들러 파일 (internal/api 안)

`internal/api` 는 HTTP 표면이자 `store`·`match` 를 부르는 자리다. 새 관측·데모
핸들러는 여기 새 파일로 짓고 `api.go` 에는 등록 줄만 는다.

```text
   nodes.go   getNodes    GET /v1/nodes        관측 스냅숏 (ADR-065 모양)
   runs.go    getRuns     GET /v1/runs (목록)   필터 넷 · limit 기본 100
   demo.go    postDemo    데모 제출 (Q5 = A)    allow-list · 토큰 서버측 주입
   (기존)     getRun      GET /v1/runs/{id}     requires · as · chosen 추가
```

**데모 제출 (demo.go · Q5 = A)**: 서버측 로직이라 `internal/api/ui`(정적)에 못
산다. 고정 시나리오 픽스처는 `internal/contract` 의 example 에서 오고
(`runctl example` · `decisions §8.3`), allow-list 는 그 픽스처 이름 집합이다.
실 토큰을 서버측에서 주입하므로 브라우저에 안 나온다 (SECURITY-08 가둠).

---

## 5. 임포트 불변식 (검사기가 표를 읽는다)

```text
   internal/panel   -> internal/store    금지
   internal/panel   -> internal/api      금지
   internal/api/ui  -> internal/store    금지
   internal/enode   -> internal/panel    금지 (cmd/enode -> panel 은 허용)
```

`internal/panel` · `internal/mcp` 가 `internal/runctl` 을 임포트하는 것은
불변식에 안 걸린다 — `runctl` 은 store·api 가 아니다. `cmd/enodectl` ·
`internal/panel` 이 `internal/proc` 를 임포트하는 것도 안 걸린다.

경계 검사 테스트는 `internal/panel` 을 만드는 유닛이 함께 낸다 — CP0 의 조건이
아니다 (그때는 검사할 패키지가 없어 항진명제다 · `constraints`).

---

## 6. 열린 미정 (자리만 남긴다)

```text
   1  제어판이 데몬 Capabilities{Caps,At} 를 읽는 계약 (ADR-068 · canon §1).
      internal/panel 에 「탐지 능력 · 잰 시각」을 내는 핸들러 자리만 두고 출처는
      TBD.  진행자가 decisions.md 에 행을 더해 해당 유닛 FD 전에 닫는다
   2  sandbox 표시 출처 (decisions §8.5).  있는 값을 읽는다.  FD 가 정한다
```
