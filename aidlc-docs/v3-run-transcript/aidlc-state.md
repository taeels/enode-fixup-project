# AI-DLC State Tracking — v3-run-transcript

## Project Information
- **Project Type**: Brownfield
- **Start Date**: 2026-09-15T03:26:50Z
- **Current Stage**: **INCEPTION 종료** (2026-09-15T08:32:42Z 승인). CONSTRUCTION 은 담당 루트에서 돈다 — `aidlc-docs/taeels/`
- **AI-DLC Version**: 1.0.1 (`.aidlc/aidlc-rules/`)
- **Run Branch**: `v3-run-transcript` (Inception 산출물이 여기 직렬로 쌓인다. CONVENTIONS.md 3.1)
- **문서 루트**: `aidlc-docs/v3-run-transcript/` (CLAUDE.md 의 회차별 layering)
- **Office Task**: `a6c3195e`

## Workspace State
- **Existing Code**: Yes
- **Programming Languages**: Go (module `github.com/taeels/enode`)
- **Build System**: Go modules. CI 는 GitHub Actions
- **Project Structure**: 멀티 바이너리 모노레포 — `cmd/{mediator,enode,enodectl,runctl,iapadapter}` 다섯 + `internal/*` 열둘. 설계 정본은 서브모듈 `enode-design/`
- **Workspace Root**: /home/sunny/enode-fixup-project
- **Go 소스**: 198 파일 (비테스트 97 · 비테스트 24,534 줄)
- **Mediator 라우트**: 26 (`internal/api/*.go` 의 `mux.HandleFunc` + `mux.Handle(`). `api.go` 한 파일의 `HandleFunc` 만 세면 17 이고 아홉을 놓친다 — 짝 팩 회차가 셈법을 고친 자리다
- **enode-design 핀**: `29c89cd` · `origin/main` 과의 거리 0
- **회차 브랜치 기준선**: `5716714` (`main` 을 합친 커밋). 짝 팩의 유닛 다섯이 이미 `main` 에 있다

## Reverse Engineering — 낡았다. 이 회차가 갱신한다
공용이다 — `aidlc-docs/inception/reverse-engineering/` (CLAUDE.md). 회차가
자기 루트에 사본을 두지 않는다.

- **마지막 전면 측정**: 2026-09-08T07:17:58Z (여덟 문서 · 비테스트 75 파일)
- **마지막 부분 갱신**: 2026-09-11T13:13:54Z — 짝 팩이 읽는 네 경로만
  (`internal/enode` · `internal/contract` · `cmd/iapadapter` · `cmd/runctl`)

**판정 — 낡았다.** `workspace-detection.md` Step 3 의 분기를 그대로 적용한다.
사용자에게 묻지 않는다. 근거는 측정값이다.

```text
   이 팩이 읽는 경로       기준선 06215ff 이후 비테스트 .go 변경 · 신규
   internal/api            8 · 7      라우트 15 -> 26.  api-documentation.md 는 12 개만 적는다
   internal/api/ui         1 · 1      산출물에 이름이 0 번 나온다.  static/ 전체가 새 표면이다
   internal/panel          5 · 5      패키지 전체가 신규.  산출물에 이름이 1 번(dependencies) 뿐이다
   internal/enode         14 · 4      2026-09-11 에 부분 갱신됨
   internal/record         0 · 0      무변경
   cmd/runctl              2 · 0      2026-09-11 에 부분 갱신됨
```

**막는 것이 무엇인가** — 이 팩의 3.3 이 `internal/panel` 의 카드를 바꾸고 3.6 이
`internal/api/ui` 에 카드를 더한다. 그 둘이 공용 산출물에 **없다.** 없는 것을
딛고 Requirements 를 쓰면 그 문서가 코드를 안 보고 쓴 것이 된다.

- **범위**: 여덟 문서 전면 갱신. 부분 갱신을 한 번 더 하면 안 잰 자리가 두 겹으로 쌓인다
- **자리**: 공용 경로 그대로. 이 회차의 루트에 사본을 안 만든다
- **완료**: 여덟 문서를 다시 썼고 방법과 못 잰 것을
  `reverse-engineering-timestamp.md` 가 적는다
- **승인**: 2026-09-15T03:56:47Z (사용자 「계속하라고」 · 「트랜스크립트 하던거」)
- **시각 하나가 측정이 아니다**: 앞 세션이 완료를 `2026-09-15T04:10:00Z` 로 적었는데
  승인 시각(03:56:47Z)보다 뒤다. 짐작한 값이고 측정한 값이 아니다. 산출물 여덟 장의
  내용은 그대로 참이다 — 시각만 근거가 없다

### 이 팩의 전제를 뒤집는 측정값 다섯
Requirements Analysis 가 이것을 입력으로 받았다. **그중 둘이 값이 아니라 결정이라
사용자에게 물었다** — 질문 파일의 1 과 2 다. 나머지 셋은 범위가 줄거나 값이 정해지는
방향이라 「확인된 사실」로 적었다.

```text
   ①  3.1 의 절반이 이미 있다      Argv 가 -p --output-format stream-json --verbose 다.
                                    남은 것은 Decode 뿐이다 (아직 io.ReadAll 배치)
   ②  링 tee 가 꺼져 있다          짝 팩 ⑲ 가 하네스 단계의 tee 를 껐다.  팩이 적은
                                    「끝에 JSON 한 줄」이 아니라 에이전트 단계 내내 0 줄이다
   ③  logs/ 가 원문이 아니다        짝 팩 ⑱ 의 허용목록.  팩의 3.2 · CB1 · CB2 · CB3 이
                                    「같은 사건 열」과 「같은 바이트」를 요구하는데 못 선다
   ④  AppendLog 가 총 길이를 안 낸다 반환값은 이번 호출의 바이트 수다.  상한도 호출마다다
   ⑤  runctl mcp 가 main 에 없다    3.7 은 이월이고 CB5 는 「해당 없음」이다
```


## Code Location Rules
- **Application Code**: 저장소 뿌리 (`cmd/`, `internal/`). aidlc-docs/ 에는 안 들어온다
- **Documentation**: `aidlc-docs/v3-run-transcript/` 만 (공용 R/E 는 예외 — 위)
- **Structure patterns**: See code-generation.md Critical Rules

## Requirements Pack (입력)
이 회차는 요구 팩이 이미 있다. Requirements Analysis 가 이것을 입력으로 읽는다.
- `requirements/transcript/features.md` — 기능 일곱 (3.1 ~ 3.7 · 필수 여섯 + 조건부 하나)
- `requirements/transcript/decisions.md` — 결정표 (1 ~ 6절). 5절이 앞 팩의 대체 문장을 센다
- `requirements/transcript/scene-gates.md` — 장면 조각 게이트 CB0 ~ CB6 (수용 기준)
- `requirements/transcript/canon.md` — enode-design 정본과의 연결
- `requirements/transcript/constraints.md` — 제외 일곱 범주 · 구조 불변식 · 접점

짝 팩은 `requirements/harness-components/` (`v3-run-harness-components`)이고
**이미 `main` 에 들어왔다.** 접점은 `constraints.md` 의 접점 절이 적는다.

## Extension Configuration
`requirements/transcript/decisions.md` §1 이 사용자 결정(2026-09-11)으로 이미
닫았다. Requirements Analysis 에서 확인만 한다 — 새로 묻지 않는다.

| Extension | Enabled | Decided At |
|---|---|---|
| security-baseline | Yes | decisions.md §1 (사용자 결정 2026-09-11) |
| resiliency-baseline | No | decisions.md §1 |
| property-based-testing | No | decisions.md §1 |

취급: `decisions.md` §3 이 SECURITY 규칙마다 처리를 미리 적었다 — 새 표면(GET
라우트 · PUT 응답 · 현황판 카드 · 파서)에만 걸고 기존 코드의 사실은 기록만 한다.

## Execution Plan Summary — Workflow Planning (2026-09-15T05:52:20Z)

전문은 `inception/plans/execution-plan.md`. 위험도 **High** · 되돌리기 Moderate ·
검사 복잡도 Complex.

```text
   실행    7   Application Design · Units Generation · Functional Design ·
               NFR Requirements · NFR Design · Code Generation · Build and Test
   스킵    1   Infrastructure Design — 새 실행파일 0 · 새 포트 0 · DB 스키마 0
   경로    6   transcript (신규) · enode · record · api · panel · api/ui
   게이트  7   CB0 ~ CB6.  기계 1 · 실패 주입 1 · 사람 눈 4 · 해당 없음 1 (CB5)
```

**짝 팩과 갈린 자리 하나 — NFR Requirements 를 돌린다.** 짝 팩이 그 자리를
스킵했고 그 회차의 계획이 스스로 규칙 위반으로 적었다. 이 회차는 Execute IF
넷 중 셋이 걸리고 (성능 · 보안 · 확장) `requirements.md` 가 안 닫은 값이
둘이다 — N1 함대 규모 · N2 진행 파일의 디스크 수명.

## Application Design — 닫힌 미결 일곱 (2026-09-15T06:20:00Z)

전문은 `inception/application-design/application-design.md` 1절.

```text
   D1  진행 파일의 자리    <Root>/progress/run-<id>/NN-<name>.log.  **기록 디렉터리 밖**
   D2  PUT 의 갈림        기존 라우트에 ?progress=1&attempt=<n>.  새 라우트 0
   D3  출처 헤더          X-Enode-Log-Source: progress | sealed
   D4  AppendLog          시그니처 그대로.  반환값의 뜻만 총 길이로
   D5  selectLogs 와 파서  껍데기를 짓는 셋을 transcript 로 옮긴다
   D6  유닛 분해          Units Generation 의 몫.  여기서 안 가른다
   D7  재시도             시도가 바뀌면 진행 파일을 비운다.  링과 같은 답
```

**D1 의 근거가 실측이다** — `seal(dir)` 이 `Walk` 로 0444 · 0555 로 굳히고
보관 정책이 없다 (`record.go:156` · `:196`). 기록 디렉터리 안에 두고 지우기를
한 번 놓치면 그 Run 의 원문이 **아무도 못 지우는 상태로 영구가 된다.**

**D7 이 파생 결정 하나를 낳았다** — 비우면 총 길이가 뒤로 간다. 링이 `gen` 으로
이미 푼 자리라 같은 답을 쓴다: 응답이 `X-Enode-Log-Attempt` 를 싣고 화면이 그
값이 바뀐 것으로 카드를 비운다.

## 이 단계가 찾은 일곱째 경로

`requirements.md` 7.3 이 여섯을 적었는데 **일곱이다.** 제어판은 Mediator 를
`runctl.Client` 로만 부르므로 (`panel.go` 패키지 주석) FR-4 의 출처 전환에
`internal/runctl` 의 클라이언트 메서드 하나가 는다. `enodectl.exe` 의 차단
게이트에는 안 걸린다 — `cmd/enodectl` 이 `internal/runctl` 을 임포트하지 않는다.

## Units Generation — 유닛 여덟 (2026-09-15T06:50:00Z)

전문은 `inception/application-design/unit-of-work*.md` 넷.

```text
   U1 transcript      FR-3           코드 게이트만.  가장 먼저 선다
   U2 node-stream     FR-1 · FR-2    코드 게이트만.  U1 을 안 기다린다
   U3 progress-store  FR-5 (med)     코드 게이트만.  **N2 를 진다**
   U4 log-api         FR-6 · FR-5    CB0.  **N1 을 진다**
   U5 panel-live      FR-4 절반      **CB1** — 이 팩의 이유
   U6 panel-past      FR-4 나머지    CB2
   U7 chunk-push      FR-5 (노드)    CB3
   U8 fleet-card      FR-7           CB4 · CB6
```

**착수 순서는 웨이브가 아니라 CB1 이 정한다** (Q3 = B) —
U1 · U2 · **U5(CB1)** · U3 · U4(CB0) · U6(CB2) · U7(CB3) · U8(CB4 · CB6).

**병합 규칙은 Q2 = A 다** — 게이트가 재는 기능을 마지막으로 완성하는 유닛이
그 게이트를 지고, 앞선 유닛은 코드 게이트로 병합한다. `CONVENTIONS.md` 3.3 을
그대로 읽으면 CB1 하나가 유닛 셋을 묶어 세워 아무것도 안 움직인다.

**배정 안 된 것 0 · 셋 다** — 스토리 열하나 · 완료 조건 여섯 · 기능 여덟.
CB5 와 FR-8 은 이월이라 유닛이 0 이다.

## 이 단계가 찾은 것 — 경계 검사 둘

```text
   ①  requirements.md 5.2 의 넷 중 **셋만 검사기에 있다**.
      api/ui -> store 는 규칙으로만 있고 검사가 없다 (boundary_test.go 가 유일한 검사)
   ②  「검사기는 표를 읽는다」가 반쯤만 참이다.  panel 의 금지만 슬라이스이고
      나머지는 if 둘이다.  transcript 의 금지 넷을 넣을 자리를 U1 이 정한다
```

## Stage Progress

### INCEPTION PHASE
- [x] Workspace Detection — 2026-09-15T03:26:50Z
- [x] Reverse Engineering — 전면 갱신 완료. 승인됨 2026-09-15T03:56:47Z
- [x] Requirements Analysis — 승인됨 2026-09-15T04:10:00Z
      질문 셋 · 답 셋(B · A · B) · `requirements.md` 536줄 · 커밋 `b87ffc9`
- [x] User Stories — 승인됨 2026-09-15T05:52:20Z (사용자 「승인. 워크플로 플랜 하자」)
      평가(`plans/user-stories-assessment.md`) · 계획과 답 넷(전부 A) ·
      페르소나 셋 · 스토리 열하나 · **새 완료 조건 여섯 (NC-1 ~ NC-6)** · 커밋 `94d28ed`
- [x] Workflow Planning — 승인됨 2026-09-15T06:05:00Z (사용자 「승인」)
      `plans/execution-plan.md` 477줄. 실행 7 · 스킵 1. 새로 찾은 것 셋(D7 · N1 N2 · 주석 넷) · 커밋 `2b0d6f2`
- [x] Application Design — 승인됨 2026-09-15T06:35:00Z (사용자 「승인」)
      계획과 답 넷(전부 A) · 산출물 다섯 931줄. **D1 ~ D7 이 전부 닫혔다** ·
      일곱째 경로 하나(`internal/runctl`) · 커밋 `9af7f3a`
- [x] Units Generation — 승인됨 2026-09-15T08:32:42Z (사용자 「승인. 이제 구축 하자」)
      계획과 답 셋(A · A · B) · 산출물 넷. **유닛 여덟** · 배정 안 된 스토리 · NC · FR 이 0 · 커밋 `93a97ad`

**Inception 이 여기서 닫혔다.** 아래 CONSTRUCTION 의 진행은 이 파일이 안 진다 —
담당 루트 `aidlc-docs/taeels/aidlc-state.md` 가 진다 (`CLAUDE.md` 의 문서 루트 규약).
이 절은 회차 계획이 무엇을 EXECUTE 로 정했는지의 기록으로만 남는다.

### CONSTRUCTION PHASE
담당은 `taeels` 하나. 문서 루트 `aidlc-docs/taeels/` · 유닛마다 `unit/<유닛>`.

**착수 배치가 웨이브 다섯이다 (2026-09-15T08:33:10Z · 사용자 「병렬로 돌릴 수
있으면 돌려」).** `unit-of-work-dependency.md` 3절의 여덟 직렬은 **한 손을 전제로**
쓴 순서이고, 의존 행렬과 파일 행렬에 다시 대면 다섯으로 준다. 전문은 담당 루트.

```text
   W-a   U1 transcript   병렬  U3 progress-store    의존 0 · 파일 겹침 0
   W-b   U2 node-stream  병렬  U4 log-api           enode 대 api.  **CB0**
   W-c   U5 panel-live   단독                       **CB1** — 뒤의 셋이 전부 딛는다
   W-d   U6 panel-past   병렬  U7 chunk-push        panel·runctl 대 enode
   W-e   U8 fleet-card   단독                       **CB4 · CB6**
```

**U1 과 U2 가 같은 웨이브에 못 든다** — 둘 다 `internal/enode/runner.go` 를 만지고
파일 행렬 2절이 `U1 -> U2` 로 못 박았다. **U5 를 단독으로 두는 것이 이 배치의 값**
이다 — CB1 을 보기 전에 뒤의 셋을 지으면 되돌릴 것이 셋이 된다.

- [ ] Functional Design — **EXECUTE** (유닛마다). 형식 다섯을 닫는다
- [ ] NFR Requirements — **EXECUTE** (유닛마다 · 최소). 안 닫힌 값 둘(N1 · N2)
- [ ] NFR Design — **EXECUTE** (유닛마다 · 최소). 패턴 다섯
- [ ] Infrastructure Design — **SKIP**. 배포 모형이 안 바뀐다. 디스크 값은 N2 로 갔다
- [ ] Code Generation — **EXECUTE** (ALWAYS · 유닛마다)
- [ ] Build and Test — **EXECUTE** (ALWAYS). CB0 ~ CB6 이 곧 시험 계획이다

### OPERATIONS PHASE
- [ ] Operations — PLACEHOLDER

## 닫힌 결정 셋 — 사용자 답 (2026-09-15T04:02Z)

`inception/requirements/requirement-verification-questions.md`. 사용자
「권장으로 결정」.

```text
   질문 1 = B   도는 동안만 본문, 봉인 때 걷는다.  진행 로그는 logs/ 가 아닌
                별도 파일이고 봉인할 때 지운다.  logs/ 는 오늘 그대로 선별본이다
   질문 2 = A   링은 원문 그대로.  ⑲ 의 넷째 길을 링에는 안 건다
   질문 3 = B   한 손(taeels).  Inception 을 돌고 그대로 Construction 까지
```

**조합이 게이트 넷 중 셋을 살리고 하나를 고친다** — CB1 은 답 2 = A 가, CB4 는
답 1 = B 가 살렸다. CB3 은 잴 것이 하나 늘고 CB2 는 고쳐 썼다 (`requirements.md` 6절).

## 이 회차가 지는 보안 잔여 셋

`requirements.md` 5.4 가 전문이다. 사용자가 값을 보고 고른 것이고 없애는 길은
게이트 CB1 · CB4 를 죽이는 길이었다.

```text
   ①  도는 동안 Mediator 디스크에 원문이 앉는다.  봉인 때 지워진다.
       봉인이 안 와도 마지막 쓰기로부터 6시간이면 걷힌다 (N2)
   ②  그동안 GET log 를 부를 수 있는 사람이 그것을 읽는다.  주체별 권한 모델이 없다
   ③  링에 원문이 노드 디스크에 앉는다.  0600 · 루프백 · 단계마다 덮인다
```

**① 의 상한이 섰다 (2026-09-15 · U3 의 NFR Requirements).** 미결로 들고 왔던
자리이고 값이 여기서 채워졌다.

```text
   무엇의 나이   진행 파일의 마지막 쓰기.  Run 이나 단계의 시작이 아니다 —
                노출은 바이트가 앉아 있는 기간이다.  끝난 단계의 파일이 여기서 걸린다
   상한         6시간.  상수다 (설정 키를 안 만든다)
   집행         Reap() 안.  sealExpired 안이 아니다 — 그쪽은 n > 0 뒤라
                구멍을 물려받는다.  새 타이머 0 (RunReaper 가 이미 60초로 돈다)
   대가         도는 Run 의 관측이 한 번 끊긴다.  본문을 잃고 카드가 비지만
                단계 · 실행 · 봉인되는 logs/ 는 안 다친다.  NC-1 의 경과 표시가
                「지워졌다」와 「죽었다」를 가른다
```

**앞 판의 「봉인 때 지워진다」가 반만 참이었던 것도 함께 닫혔다** — U3 이 구멍
둘을 코드로 짚었다 (`reap.go:126` 의 한 시간 창 · `:104` 의 `n > 0`). 셋째로
적었던 `s.Records == nil` 은 **구멍이 아니다** — `needRecords` 가 503 을 내
진행 트리가 애초에 안 생긴다.

**남는 것이 하나 있다 — 디스크다.** 6시간이 자르는 것은 노출 시간이지 디스크가
아니라서, Run 이 6시간보다 짧게 돌면 최악의 봉투(R=50 · S=20 이면 10 GiB)가
그대로 선다. 사용자가 그 값을 보고 **경보와 총량 상한을 안 만드는 쪽**을 골랐다
(`freeBytes` 가 노드 쪽에만 있어 들이면 같은 함수가 두 벌이 된다).

## 회차가 넓히는 자리 하나

`requirements.md` 5.5 — 제어판 페이지(`internal/panel/page.go:15`)에 `/ui/` 와
같은 보안 헤더 다섯을 건다. 이 회차가 그 페이지에 신뢰할 수 없는 하네스 출력을
그리므로 위험을 들여오는 쪽이 닫는다. 빼려면 그 절을 지우고 SECURITY-04 판정을
다시 센다.

## 이 단계가 낳은 새 완료 조건 여섯 — 하류가 받는다

`inception/user-stories/user-stories.md` 3절이 전문이다. Q4 = A 에 따라
**스토리가 지고 `requirements.md` 는 안 고친다.** Units Generation 이 유닛에 내린다.

```text
   NC-1   마지막 사건 이후 경과가 보인다.  침묵과 정지가 갈린다              US-1
   NC-2   링이 감겨 앞이 잘렸음이 적힌다.  total 은 이미 나오는데 안 그린다   US-2
   NC-3   링에 원문이 남는다는 사실이 제어판 카드에 한 줄로 적힌다           US-3
   NC-4   진행 파일이 상한에 닿았음이 응답과 화면에 적힌다                   US-7
   NC-5   mediator-api 의 GET log 절이 봉인 전후의 갈림을 적는다             US-8
   NC-6   현황판 카드에 마지막 갱신 시각이 보인다.  밀림과 정지가 갈린다      US-9
```

**여섯 중 다섯이 같은 모양이다** — 「안 자라는 것」과 「끝난 것」을 사람이 구별할
수 있는가. 게이트 조각 일곱이 전부 「자란다」를 재고 「안 자란다」를 재는 것이 0 이다.
