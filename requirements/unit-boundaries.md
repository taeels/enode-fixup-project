# 권장 유닛 경계 — 유닛은 계획 단위다. 배포 단위가 아니다

v1 의 Units Generation 은 파일 경로를 유닛에 적지 않는다. 그래서 여기 미리
적는다. **유닛마다 만지는 파일이 있고, 두 유닛이 같은 파일을 만지면 그 자리가
병목이다.** v1 이 이 경계를 그대로 받아도 되고 더 잘 갈라도 된다 — 다만
**파일 행렬은 반드시 낸다.**

```text
   유닛 = 브랜치 하나          unit/바닥 처럼 (START.md §5)
   병합 순서 = 착수 순서        먼저 시작한 것이 먼저 들어간다
   api.go 는 등록 줄만          새 핸들러는 새 파일.  등록 줄 병합은 직렬
   새 기능은 새 패키지          유닛끼리 서로 임포트하지 않는다.  CP0 가 기계로 센다
   겉면은 얇게                  Config · Server · New.  화면과 제어판은 관측 API 를
                               HTTP 로 부른다.  store 를 직접 안 부른다
```

---

# 1. 유닛 일곱

| 유닛 | 하는 것 | 요구사항 절 | 끝났다는 것 | 딛는 것 |
|---|---|---|---|---|
| U-바닥 | 스키마 · 상태 상수 · 조회 메서드 · 정책 자리. **0일차 직렬.** 다른 유닛이 전부 이것을 딛는다 | 3.2.1 `QUEUED` · 3.2.2 정책 · 3.1.1 조회 | 기존 테스트 전부 초록 + 새 열과 상수가 있고 기존 라우트가 그대로 돈다 | 없음 |
| U-관측API | `GET /v1/runs` · `GET /v1/nodes` (+ `StepView.chosen`) | 3.1.1 | curl 로 두 라우트가 실데이터 JSON. 임대와 `draining` 이 실려 있다 | U-바닥 |
| U-대기열 | `409` -> `QUEUED` · FIFO · `WakeQueued` · `202` · 기동 시 깨우기 | 3.2.1 | 노드 하나에 Run 둘 -> 둘째 `QUEUED` -> 첫째 끝나면 승격 | U-바닥 |
| U-drain | 정책 파일 읽기 · 광고에 싣기 · 후보 제외 · 응답 통보 · 두 모드 · 해제 | 3.2.2 | `at-boundary` 로 걸면 경계에서 풀리고 대기 Run 이 집는다. 풀면 후보로 돌아온다 | U-바닥 · U-대기열(깨우기) |
| U-제어판 | `enodectl serve` · 프로세스 제어 · 신원 · 탐지 능력 · 현재 작업 · drain 토글 | 3.1.2 | 도는 노드에서 열면 임대 · 단계 · 회차가 보인다. 멈춘 노드에서 `start` 가 된다. `127.0.0.1` 밖에서 안 붙는다 | U-관측API · U-drain(정책 파일 형식) |
| U-화면 | 중앙 현황판 정적 파일 (S0 · S0b · S1 · S1b · S2) · 호스트 화면 (S3 · S5) | 3.1.1 · 3.1.2 · `design/` | `design/exports/` 의 장면이 실데이터로 뜬다 | U-관측API · U-제어판 |
| U-MCP | `runctl mcp` — stdio JSON-RPC · 도구 여덟. 기존 REST 를 감싼다 | 3.3.1 | 사용자 Claude 가 붙어 `run.submit` 한 Run 이 현황판에 뜬다. `fleet.list` 가 `GET /v1/nodes` 와 같은 글자다 | U-관측API (라우트만) |

**U-화면은 SKIP 가능 축이다.** 폭이 있으면 따로 가르고, 없으면 U-관측API 와
U-제어판에 각각 합친다. 합칠 때도 파일은 `ui/` 아래로 분리한다.

**U-바닥이 늦으면 전원이 선다.** 1차 배치 폭이 1 이다. 그래서 0일차에 한
사람이 직렬로 끝내고, CP0 가 초록이어야 1일차가 시작된다.

---

# 2. 유닛이 만지는 파일 — 겹침 검사

`o` 는 만진다, `c` 는 호출만 하고 고치지 않는다, `NEW` 는 새 파일이다. 한 열에
`o` 가 둘 이상이면 그 파일이 접점이고, 3절에 처리가 있다.

```text
   파일                                                          바닥  관측  대기열  drain  제어판  화면  MCP
   ───────────────────────────────────────────────────────────── ───── ───── ────── ────── ─────── ───── ───
   internal/store/schema.sql                                     o
   internal/store/store.go  (UpsertAdvert)                       o                  o
   internal/store/observe.go  (Chosen)                           o
   internal/store/list.go                   NEW                  o     o
   internal/store/queue.go                  NEW                  o           o
   internal/store/release.go  (부분 반납 뒤 깨우기)                          o
   internal/store/reap.go  (Reap 뒤 깨우기 · Cancel)                         o      c
   internal/contract/advert.go  (Policy)                         o                  o
   internal/api/api.go  (등록 줄 · submit busy · 광고 응답 · postResult 종료)            o     o      o              .
   internal/api/list.go                     NEW                        o
   internal/api/queue.go                    NEW                              o
   internal/api/ui/                         NEW                                                    o
   cmd/mediator/main.go  (기동 시 깨우기)                                    o
   internal/enode/policy.go                 NEW                                     o
   internal/enode/advertise.go  (정책 싣기)                                         o
   internal/enode/claim.go  (Worker.SetDrain)                                       o
   cmd/enode/main.go  (OnPolicy · case "panel")                                     o      o
   internal/panel/                          NEW                                            o
   cmd/enodectl/serve.go                    NEW                                            o
   cmd/enodectl/main.go  (case "serve" · 도우미 내리기)                                    o
   cmd/enodectl/proc_unix.go · proc_windows.go  (processAlive)                             o
   cmd/enodectl/setup.go  (enodeBin -> panel.EnodeBin)                                     o
   .coverage-contract.yml  (platform 주석)                                                 o
   internal/mcp/                            NEW                                                          o
   internal/runctl/client.go  (Nodes · Runs 두 메서드)                                                    o
   cmd/runctl/main.go  (case "mcp" 한 줄)                                                                 o
```

**`MCP` 열은 다른 어느 열과도 안 겹친다.** 세 줄 전부 `o` 가 하나뿐이고, 그
셋을 다른 유닛이 아무도 안 만진다. 그래서 U-MCP 는 접점이 0 이고 병합 순서에
자리를 안 잡는다 — 어느 지점에 들어와도 남의 diff 위에 안 얹힌다.

`.` 은 등록 줄이 필요할 수 있으나 정적 파일 마운트 한 줄이라 직렬 병합으로
받는다는 뜻이다. `internal/match/match.go` 는 **아무 유닛도 안 만진다** — 순수
함수이고(`ADR-014`) `busy` 는 `api.go` 의 `submit` 이 만들어 넘긴다. 파일 이름은
코드베이스의 어휘를 따랐고 새 이름은 v1 이 바꿔도 된다 — 행렬이 있다는 것이
요구다.

## 2.1 U-트랜스크립트 — 2026-09-06 에 더해진 일곱째

**위 행렬을 넓히지 않고 따로 적는다.** 그 행렬은 **동시에 도는 여섯**의 겹침을
보려고 만든 것이고, 이 유닛은 그 여섯이 전부 커밋된 뒤에 **직렬로** 온다.
겹침을 물을 상대가 브랜치가 아니라 이미 병합된 코드다.

```text
   파일                                          무엇을
   ───────────────────────────────────────────── ──────────────────────────────
   internal/enode/transcript.go        NEW       링 파일.  머리 + 원형 몸통
   internal/enode/runner.go                      에이전트 단계의 stdout 을 tee
   internal/enode/claim.go                       명령 단계의 stdout 을 tee ·
                                                 단계 끝에 머리를 비운다
   internal/panel/transcript.go        NEW       GET /api/transcript?node=   (도는 것)
   internal/panel/history.go           NEW       GET /api/runs?node=          (지난 것 목록)
                                                 GET /api/runs/{id}/transcript (tar 에서 로그)
   internal/panel/panel.go                       라우트 등록 세 줄
   internal/panel/ui/assets/index.html           카드를 비활성에서 살린다 · 이력 자리
   internal/panel/ui/assets/status.js            도는 것 카드를 그린다
   internal/panel/ui/assets/history.js NEW       지난 Run 목록과 상세
   internal/panel/ui/assets/app.css              그 둘의 자리
   internal/panel/ui/assets/app.js               1초 타이머의 자리.  타이머 둘이 이미 있다
   internal/panel/ui/ui_test.go                  「트랜스크립트 버튼이 비활성」 단언을
                                                 반대로 뒤집는다.  안 고치면 게이트가 빨갛다
   internal/enode/transcript_test.go   NEW
   internal/panel/transcript_test.go   NEW
   internal/panel/history_test.go      NEW
```

**표를 2026-09-06 에 넷 늘렸다.** 앞의 둘은 이 문서의 누락이었고
(`frontend-components.md` 1절이 `app.js` 를 「수정」으로 이미 적었다), 뒤의 셋은
시험 파일이라 생산 파일만 세던 처음 판에서 빠졌다. **시험 없이는 커버리지
하한을 못 넘는다.**

**`ui_test.go` 의 단언 하나가 이 유닛으로 거짓이 된다.** U-제어판이 「S4 는
범위 밖이니 그 버튼이 아무 일도 안 해야 한다」로 넣은 것인데, 범위 결정이
바뀌어서 뒤집힌다. **지우지 않고 반대 단언으로 바꾼다** — 그래야 그 자리에
시험이 계속 선다.

**지난 것 쪽은 Mediator 를 안 고친다.** `GET /v1/runs` 응답의 `assigned` 가
노드를 이미 실어서 제어판이 걸러내면 되고, 봉인된 트랜스크립트는
`GET /v1/runs/{id}/record` 의 tar 에서 `logs/NN-<step>.log` 만 꺼낸다.
`archive/tar` 는 표준 라이브러리라 **새 Go 의존이 0** 이다
(`decisions.md` 6.5).

**위 행렬의 어느 열과도 안 겹친다.** `internal/enode/runner.go` 와
`internal/enode/claim.go` 는 행렬에 아예 없다 — 여섯 중 아무도 안 만졌다.
`internal/panel` 은 U-제어판이 만들었고 **이미 커밋됐다**(`885d868`) 이므로
병합 충돌이 아니라 그냥 이어 쓰는 것이다.

**Mediator 쪽은 0 이다.** `internal/api` 도 `internal/store` 도 `cmd/mediator`
도 안 건드린다. 트랜스크립트는 노드 밖으로 안 나간다 (`decisions.md` 6.4).

### 접점 하나

```text
   링 파일의 형식    U-트랜스크립트가 정본이다 (internal/enode/transcript.go).
                    제어판은 그 패키지의 읽기 함수를 부른다.  자기 파서를
                    안 만든다 — U-drain 의 정책 파일과 같은 처리다 (3절)
```

`internal/enode -> internal/panel` 금지는 그대로다. 방향은 여전히 제어판이
`internal/enode` 를 부르는 쪽 하나다 (`NFR-P4`).

---

---

# 3. 겹치는 곳과 처리

```text
   store.go UpsertAdvert       바닥이 정책 열을 받는 시그니처까지 만든다.
                               drain 은 값을 채우기만 한다.  순서 = 바닥 -> drain

   store/queue.go              바닥이 파일과 DrainingNodes · ListQueued 를 만든다.
                               대기열이 WakeQueued 를 더한다.  같은 파일이지만
                               함수가 다르다.  순서 = 바닥 -> 대기열

   reap.go                     대기열이 Reap 뒤의 WakeQueued 호출을 더한다 (파일을
                               만진다).  drain 은 at-boundary 의 종료 경로로 Cancel 을
                               호출만 하고 고치지 않는다

   api.go submit 의 busy       대기열이 submit 이 BusyNodes 로 busy 를 만드는 자리에
                               DrainingNodes 를 합친다.  match.go 는 아무도 안 만진다
                               — 순수 함수다 (ADR-014).  DrainingNodes 는 바닥의 조회다

   api.go 등록 줄               관측 · 대기열 · 화면 각 한 줄.  drain 은 광고 응답에
                               drain 값을 더하는 줄 하나와 postResult 끝의 at-boundary
                               종료(Cancel 호출) 한 블록.  병합 순서대로 직렬

   정책 파일 형식               drain 이 정본(internal/enode/policy.go).  제어판은
                               그 패키지의 Read · Write 를 쓴다.  자기 파서를 안 만든다

   현재 작업의 출처             제어판은 Mediator 조회(GET /v1/nodes · GET /v1/runs/{id})
                               로 조립한다.  decisions.md 2절.  데몬을 안 건드린다

   cmd/enode/main.go           drain 이 adv.OnPolicy = worker.SetDrain 한 줄을 더했고
                               제어판이 case "panel" 한 블록을 더한다.  자리가 다르다 —
                               앞은 데몬 조립, 뒤는 플래그 파싱 앞의 이른 갈래.
                               drain 이 먼저 커밋됐으므로 직렬로 받는다.
                               순서 = drain -> 제어판

   cmd/enodectl/setup.go       제어판만 만진다.  enodeBin() 이 internal/panel 로
                               내려가므로 부르는 자리를 panel.EnodeBin() 으로 바꾼다.
                               setup 의 위임 구조 자체는 안 건드린다

   .coverage-contract.yml      제어판만 만진다.  platform 주석이 윈도우 파일 셋을
                               이름으로 적었고 그중 proc_windows.go 가 옮겨 간다.
                               주석만 고치고 수치를 안 고친다
```

## 3.1 제어판이 `cmd/enode/main.go` 에 착지하는 이유

**`enodectl serve` 가 HTTP 서버를 직접 부르면 CI 가 막는다.** `ci.yml` 의
`cross` 잡이 `enodectl.exe` 의 `crypto/tls` 코드 심볼 10 개 · `net/http` 코드
심볼 50 개를 상한으로 걸고 있고 `continue-on-error` 가 없다. 지금 값은 1 과
6 이고 `enode.exe` 는 393 과 687 이다 — `internal/panel` 의 서버를 `enodectl`
의 `main` 에서 부르면 링커가 스택을 통째로 넣어 상한을 39배 · 14배로 넘는다.

그 상한은 취향이 아니다. `v0.0.1-rc13` 의 `enodectl.exe` 가 사용자 기계에서
백신에 삭제된 사건의 흔적이고 경위는 `scripts/avprobe/README.md` 에 있다.

그래서 `enodectl serve` 는 **위임**이다 — `enode panel` 을 실행하고 표준
입출력 셋을 잇고 종료 코드를 그대로 쓴다. `cmd/enodectl/setup.go` 가 이미
같은 방법을 쓰고 있고 `ci.yml` 의 실패 문구가 그 파일을 가리킨다. 결정표는
안 깨진다 — 실행파일이 여전히 넷이고 `packaging/` 을 안 건드린다. 금지된
임포트 방향은 `internal/enode -> internal/panel` 하나이므로(`NFR-P4`)
`cmd/enode` 가 `internal/panel` 을 임포트하는 것은 규칙 안이다.

**진행자 승인 2026-09-06.**

---

# 4. 접점 표 — 착수 전에 채운다

유닛 둘이 만나는 자리마다 **제공하는 쪽 · 쓰는 쪽 · 먼저 합의할 것**을 적는다.
아래는 시작값이고, Units Generation 이 확정한다.

| 접점 | 제공 | 사용 | 먼저 합의할 것 |
|---|---|---|---|
| `contract.Policy{Drain}` | U-바닥 | U-drain · U-관측API · U-제어판 | 값 셋 `""` · `graceful` · `at-boundary`. 문자열 그대로 |
| `store.DrainingNodes` | U-바닥 | U-대기열 · U-drain | 반환 `map[string]bool` — `match.Match` 의 `busy` 와 같은 모양 |
| `store.ListRuns` · `store.ListNodes` | U-바닥 | U-관측API | 필터 인자와 정렬. `decisions.md` 2절의 응답 모양 |
| `store.WakeQueued(ctx) error` | U-대기열 | U-drain(해제) · 기동 | 호출 지점은 `decisions.md` 1절 깨우기 행. 재진입 · 동시 호출 안전 — FIFO 는 DB 의 `created_at` 순서로 |
| `store.UpsertAdvert` 의 반환값 | U-바닥 | U-drain | 이전 정책을 돌려준다. 광고 처리가 「해제됐다」(`at-boundary`/`graceful` -> `""`)를 이 값으로 감지해 `WakeQueued` 를 부른다 |
| `GET /v1/nodes` 응답 | U-관측API | U-제어판 · U-화면 | `ADR-065` 모양 + `draining` |
| 정책 파일 경로 · 형식 | U-drain | U-제어판 | `enode.ReadPolicy(path)` · `enode.WritePolicy(path, enode.Policy)`. 필드는 `drain` 과 `panel_token` — 후자는 제어판만 읽는다 |
| `enodectl serve` 의 라우트 | U-제어판 | U-화면 | `/api/status` · `/api/policy` · 정적 `/` |

---

# 5. CP0 가 기계로 세는 것

U-바닥의 산출물에 **경계 검사 테스트**를 넣는다. `go/parser` 로 각 패키지의
임포트를 읽어 다음을 확인한다.

```text
   internal/panel  ->  internal/store        금지.  제어판은 HTTP 로 Mediator 를 본다
   internal/panel  ->  internal/api          금지
   internal/api/ui ->  internal/store        금지.  화면은 정적 파일이다
   internal/enode  ->  internal/panel        금지.  데몬은 제어판을 모른다
```

이 테스트가 CP0 에 들어가면 겹침을 사람이 안 세도 된다. 패키지 이름이 바뀌면
표도 같이 바꾼다 — 검사기는 표를 읽는다.
