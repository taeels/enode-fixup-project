# AI-DLC State Tracking — v4-run-finalize-bake

## Project Information
- **Project Type**: Brownfield
- **Start Date**: 2026-09-23T13:49:57Z
- **Current Stage**: CONSTRUCTION — U5 merge-rules 의 Code Generation (계획 · Part 1) (브랜치 `unit/merge-rules` · main `b44b46e` 에서 땄다). U4 trash 는 PR #64 로 병합 2026-09-26T15:34:52Z · U3 finalize 는 PR #63 · U2 step-phase 는 PR #62 · U1 contract-grammar 는 PR #61. Inception 은 2026-09-24T12:31:50Z 에 닫혔다
- **AI-DLC Version**: 1.0.1 (`.aidlc/aidlc-rules/`)
- **Run Branch**: `v4-run-finalize-bake` (Inception 산출물이 여기 직렬로 쌓인다. CONVENTIONS.md 3.1)
- **문서 루트**: `aidlc-docs/v4-run-finalize-bake/` — Inception 과 Construction 모두 (CLAUDE.md 의 회차별 layering). Construction 은 `construction/` 아래. 2026-09-24 에 규약을 이렇게 바꿨다 — 처음 판은 Construction 을 `aidlc-docs/taeels/` 에 두었다
- **Office Task**: `f806768d` (회차 README 가 지목한다. 설계 검토와 같은 원장)

## Workspace State
- **Existing Code**: Yes
- **Programming Languages**: Go (module `github.com/taeels/enode`)
- **Build System**: Go modules. CI 는 GitHub Actions
- **Project Structure**: 멀티 바이너리 모노레포 — `cmd/{mediator,enode,enodectl,runctl,iapadapter}` 다섯 + `internal/*` 열여섯. 설계 정본은 서브모듈 `enode-design/`
- **Workspace Root**: /home/sunny/enode-fixup-v4 (git worktree)
- **Go 소스**: 263 파일 (비테스트 132 · 비테스트 31,551 줄)
- **Mediator 라우트**: 27 (`internal/api/*.go` 의 `mux.HandleFunc` + `mux.Handle(`)
- **enode-design 핀**: `a2c4ac6` (enode-design `main`. #15 가 `unit/runtime-environment-profile` 을 병합 커밋으로 합쳤다. `369270a` 와 나무가 같다)
- **회차 브랜치 기준선**: `main` `826b40f` (#59 — 실행 환경 구현이 main 에 들어갔다). 회차 브랜치가 그것을 합쳤다(`4facffb`, 나무 변화 0). R/E 의 측정 기준 `195a5d0` 과 코드가 같다

## Reverse Engineering — 낡았다. 이 회차가 갱신한다
공용이다 — `aidlc-docs/inception/reverse-engineering/` (CLAUDE.md). 회차가
자기 루트에 사본을 두지 않는다.

- **마지막 전면 측정**: 2026-09-15 (`v3-run-transcript` 회차 · 기준선 `5716714`)

**판정 — 낡았다.** `workspace-detection.md` Step 3 의 분기를 그대로 적용한다.
사용자에게 묻지 않는다. 근거는 측정값이다.

```text
   기준선 5716714 -> 195a5d0     Go 파일 131 변경 (신규 57 · 수정 42 중 비테스트 신규 28)
                                 16,262 줄 추가 · 632 줄 삭제
   패키지                         18 -> 21.  internal/environment · internal/transcript ·
                                 internal/transcriptui 가 신규
   이 팩이 딛는 파일              산출물에 나온 횟수
     internal/environment/*       0
     runc_overlay_linux.go        0
     StepRuntime                  0
   라우트                         26 -> 27
```

**막는 것이 무엇인가** — 이 팩의 접점 표(`constraints.md` 3절)가
`internal/enode/runc_overlay_linux.go` 와 `internal/environment/check.go` 를 만진다.
그 둘이 공용 산출물에 **없다.** 없는 것을 딛고 Requirements 를 쓰면 그 문서가 코드를
안 보고 쓴 것이 된다.

- **범위**: 여덟 문서 전면 갱신. 부분 갱신은 안 잰 자리를 두 겹으로 쌓는다(앞 회차의 판단을 잇는다)
- **자리**: 공용 경로 그대로. 이 회차의 루트에 사본을 안 만든다
- **완료**: 2026-09-23T14:03:35Z. 여덟 문서를 다시 썼고 방법과 못 잰 것을
  `reverse-engineering-timestamp.md` 가 적는다. **승인 2026-09-23T14:12:44Z**
- **측정**: `go build` · `go vet` exit 0. `go test ./... -count=1 -coverpkg=./...` exit 0 ·
  실패 0 · 스킵 0 · 85.3% · 스무 패키지 전부 80% 이상. 측정이 `cmd/enodectl/probe.lock` 을
  바꿔 되돌렸다 (알려진 부채)

### 이 팩의 전제와 어긋나는 측정값
Requirements Analysis 가 이것을 입력으로 받는다. 값이 아니라 결정인 것은 질문으로 간다.

```text
   ①  min_free_gb 가 가리는 것이 좁다     워크스페이스 여유만 재고, 모자라면 arch · arch.<이름>
                                         키만 뺀다 (detect.go hasRoom).  노드가 광고에서 통째로
                                         빠지는 기제가 없고 scratch 는 안 잰다.
                                         팩 기능 4 · 결정 2-6 의 「새 기제는 없다」가 서지 않는다
   ②  결과 보고가 인스턴스를 안 본다       ReportStep 이 본문의 node 와 state='CLAIMED' 로만 거른다.
                                         팩 보안 표의 「claim 한 노드와 인스턴스만」은
                                         기존 result 에도 없는 대조다
   ③  두 나무가 environment 에서 만난다    store 가 execenv.Record 타입 하나로 internal/environment
                                         를 물어 Mediator 가 그 패키지를 링크한다.
                                         boundary_test 의 금지 표에 environment 가 없다
   ④  internal/enode 의 여유가 얇다        81.5% (하한 80).  runc_overlay_linux.go 60.8% ·
                                         overlay_linux.go 48.4%.  실제 namespace 경로는
                                         integration 태그 시험만 재고 CI 밖이다.
                                         팩의 기능 4 · 6 · 7 · 10 이 이 패키지에 문장을 더한다
   ⑤  runRoot 를 지우는 자리가 셋이 아니다   팩은 :1017 · :437 · :451 셋을 적는다.  Open 의 실패
                                         갈래(:143 ~ :170)와 준비도 smoke(:1212)가 더 있다
   ⑥  정본이 코드보다 늦은 자리 하나 더     ADR-071 이 「결정 · 미구현」인데 코드는 70d6258 에서
                                         구현했다.  팩 decisions 7절은 ADR-073 E5 만 적는다
```

확인된 사실(팩과 맞는 것): `claim.go:691-694` 의 수확 인자, 진행 조회와 결과와 `steps`
표에 phase · 종료 시각이 없음, `schema.sql` 이 기준선 뒤로 안 바뀜, `ir` · `repo.built` ·
`workspace.writes` 광고 없음, 계약에 `effect` · 예산 · `sync` · `builds[]` · merge kind 없음.

## Code Location Rules
- **Application Code**: 저장소 뿌리 (`cmd/`, `internal/`). aidlc-docs/ 에는 안 들어온다
- **Documentation**: `aidlc-docs/v4-run-finalize-bake/` 만 (공용 R/E 는 예외 — 위)
- **Structure patterns**: See code-generation.md Critical Rules

## Requirements Pack (입력)
이 회차는 요구 팩이 이미 있다. Requirements Analysis 가 이것을 입력으로 읽는다.
- `requirements/finalize-bake/features.md` — 기능 열셋 · 보안 표 · 수용 기준
- `requirements/finalize-bake/decisions.md` — 결정표 (0 ~ 7절). 1-1 ~ 3-22 · 기각과 순연 4-1 ~ 4-14
- `requirements/finalize-bake/scene-gates.md` — 장면 셋 · 조각 0 ~ 12 (수용 기준)
- `requirements/finalize-bake/canon.md` — enode-design 정본과의 연결
- `requirements/finalize-bake/constraints.md` — 제외 · 구조 불변식 · 접점 · 코드 기준선

## Extension Configuration
Requirements Analysis 의 질문 4 ~ 6 이 정했다 (2026-09-23T14:35:56Z). 셋 다 꺼져 있으므로
전체 규칙 파일을 싣지 않는다.

| Extension | Enabled | Decided At |
|---|---|---|
| security-baseline | No | Requirements Analysis 질문 4 = B. 팩의 보안 표(features.md 3절)는 팩의 요구로 남는다 |
| resiliency-baseline | No | Requirements Analysis 질문 5 = B |
| property-based-testing | No | Requirements Analysis 질문 6 = X 「기존 테스트 컨벤션을 따른다」 |

## Execution Plan Summary
`inception/plans/execution-plan.md` (2026-09-23T23:57:36Z · HEAD `7a7ec85`).

- **실행**: Application Design · Units Generation · Functional Design · NFR Requirements(최소) ·
  NFR Design(최소) · Code Generation · Build and Test
- **스킵**: Infrastructure Design — 새 실행파일 · 포트 · 클라우드 자원 0.  새 디스크 자리는
  노드 사용자의 scratch 와 home 아래이고 steps 칸은 자료 모형이다
- **위험도**: High.  합치기는 되돌리기 Difficult — 시험은 버려도 되는 lower 에서만
- **이 단계가 찾은 넷**: 업로드 예산과 요청마다의 30초(`cmd/enode/main.go:206`) · 두 시계 ·
  env check 의 자리가 임포트 경계에 걸림 · 크로스 빌드 셋(linux/arm 포함)

## Application Design Summary
`inception/application-design/` 다섯 (2026-09-24T06:39:39Z). 답 일곱 다 A — 계획 파일 6절.

- **새 패키지 셋**: `internal/lower` · `internal/merge` · `internal/scratch`. 표준 라이브러리와 x/sys 만 · Mediator 가 링크 안 함
- **lower 공유 잠금**: 노드가 매칭 후보인 동안 쥔다 (Q1). 놓는 울타리는 Functional Design
- **사람이 읽는 자리**: 노드 소유자는 상태 파일과 제어판 (Q3) · 굽기 담당은 merge 단계 로그와 광고 키 bake.run (Q4) · 계약 작성자는 result 진단과 단계 로그 (Q5)
- **Mediator**: steps 칸 셋(phase · phase_since · exit) · POST .../exited 하나 · QUEUED 요구 줄의 후보 셋. match 불변
- **정정**: SunnyVM 의 /srv 는 sunny 소유다. 결정 3-14 는 다른 근거 둘로 선다 (원장 EN-bb1a4a28)

## Units Generation Summary
`inception/application-design/unit-of-work*.md` 넷 (2026-09-24). 답 일곱과 결정 둘 — 계획 파일 5절 · 2.2 · 2.3.

- **유닛 여덟 · 한 줄 순서**: contract-grammar · step-phase · finalize · trash · merge-rules · lower-state · bake · checkpoint
- **병합**: 조각은 그 기능을 마지막으로 완성하는 유닛이 맡는다. 앞선 유닛은 코드 검사로 병합
- **범위에서 뺀 것**: FR-11 결과 adapter 둘과 장면 4 (순연 4-16) · FR-12 · FR-13 은 유닛 없이 Build and Test
- **앞 단계를 고친 것**: agent 단계의 전체 훑기를 끈다 (Q7) · 굽기 계약이 구울 IR 을 값으로 적는다 (2.3 · 제품에 사내 태그 형식 없음) · 계약 작성 도구와 굽기 Run 판정은 contract-grammar 가 닫는다 (2.2)
- **원칙**: 제품 저장소에 사내 스크립트 · 구성 이름을 적지 않는다. 제품의 굽기 예시는 공개 도구로

## Construction — 유닛 여덟 · 한 줄 순서 (Units Generation Q4 = B)

혼자 도므로 병렬 웨이브가 없다. 앞 유닛이 병합된 뒤 다음 유닛의 브랜치를 회차 브랜치
(또는 병합된 main)에서 딴다. 산출물은 이 폴더의 `construction/` 아래다.

```text
   순서  유닛                맡는 조각          병합 조건
   1    contract-grammar    없음              코드 검사
   2    step-phase          0                 조각 0
   3    finalize            1 · 2 · 3         조각 1 · 2 · 3
   4    trash               4                 조각 4
   5    merge-rules         7 의 기계 부분      코드 검사 + 재개 시험
   6    lower-state         없음              코드 검사
   7    bake                5 · 6 · 7 · 8     조각 5 · 6 · 7 · 8
   8    checkpoint          9                 조각 9
```

조각은 장면 게이트의 검증 단위다 (`requirements.md` 6절). 코드 검사는 빌드 · 기본 `go test` ·
패키지별 커버리지 80% 이상 · 코드 경계 시험 · 크로스 빌드 셋이다 (`unit-of-work.md` 0절).

### U1 `contract-grammar` — 브랜치 `unit/contract-grammar` (`1367f0b` 에서 맞췄다)

```text
   Functional Design      착수 2026-09-24T12:45:25Z.  계획과 물음 일곱
                          construction/plans/contract-grammar-functional-design-plan.md
                          답 2026-09-25T03:56:21Z — A · A · A · A · A · B · C
                          되물음 셋 (답 3 과 6 의 충돌 · 계획이 지은 굽기의 승인 ·
                          자리표시 예시) — ...-clarification-questions.md.  답 A · A · A
                          (「권장대로.」).  산출물 셋 2026-09-25T04:08:26Z — construction/contract-grammar/
                          functional-design/.  승인 2026-09-25T05:03:44Z (「다음으로」)
   NFR Requirements       건너뛴다 (유닛 정의 — 성능 표면이 없다)
   NFR Design             건너뛴다
   Code Generation        Part 1 착수 2026-09-25T05:03:44Z.  계획 — 단계 열다섯
                          construction/plans/contract-grammar-code-generation-plan.md
                          설계에 없던 자리 하나 (계획 훅의 그루터기 · 계획 3절).  승인 2026-09-25T05:14:18Z.
                          Part 2 완료 2026-09-25T05:29:37Z — 단계 열넷 체크 · 코드 검사 초록 (통과 1,891 · 실패 0 ·
                          스킵 0 · 패키지 전부 80% 이상 · 크로스 빌드 셋).  요약은
                          construction/contract-grammar/code/code-summary.md.  승인 2026-09-25T05:45:28Z
   병합                   회차 PR #60 (cff1035) 뒤 PR #61 로 main 에 병합 (9b40cd0 · 2026-09-25T07:21:09Z)
```

### U2 `step-phase` — 브랜치 `unit/step-phase` (`9b40cd0` 에서 땄다)

```text
   Functional Design      착수 2026-09-25T07:21:53Z.  U1 이 넘긴 일 하나를 받는다 — Claimed 에 계약의 새 칸 일곱.
                          계획과 물음 일곱 2026-09-25T07:27:18Z — construction/plans/step-phase-functional-design-plan.md.
                          답 2026-09-25T08:30:40Z — A · A · A · A · C · A · A.  답 5 = C 가 연 자리 넷을 되물음
                          ...-clarification-questions.md 로 물었다.  답 모두 A (「모두 권장대로」).
                          산출물 셋 2026-09-25T08:42:20Z — construction/step-phase/functional-design/.
                          승인 2026-09-25T08:52:58Z (「승인한다. nfr 건너뛰고 다음으로.」)
   NFR Requirements       건너뛴다 — 사용자 결정 2026-09-25T08:52:58Z.  실행 계획은 「한다 (최소)」였다.
                          보려던 둘 중 종료 보고의 인스턴스 대조는 Functional Design 의 수락 표가 정했고,
                          대기 조회마다 매칭 두 번의 비용은 Code Generation 계획이 시험 자리로 받는다
   NFR Design             건너뛴다 (위와 같은 결정)
   Code Generation        Part 1 착수 2026-09-25T08:52:58Z.  계획 — 단계 열여섯
                          construction/plans/step-phase-code-generation-plan.md
                          건너뛴 NFR 이 보려던 둘을 계획 3절이 받는다 · FD 에 없던 자리 다섯 (계획 4절).  승인 2026-09-25T09:32:03Z.
                          Part 2 착수 2026-09-25T09:32:03Z.  완료 2026-09-25T09:48:09Z — 단계 열여섯 중 열다섯과 반 체크
                          (커밋은 승인 뒤) · 코드 검사 초록 (통과 1,960 · 실패 0 · 스킵 0 · 패키지 전부 80% 이상 ·
                          크로스 빌드 셋 · 린트 39 건 main 과 같음) · 조각 0 초록 (라우트 18 -> 19).
                          대기 조회 비용 약 0.75 ms (광고 50 · 요구 줄 3).  요약은
                          construction/step-phase/code/code-summary.md.  승인 2026-09-25T11:23:52Z (「승인」)
   병합                   PR #62 (CI 초록 · 린트 39 건 main 과 같음) 로 main 에 병합 (0c0370c · 2026-09-25T11:44:43Z)
```

### U3 `finalize` — 브랜치 `unit/finalize` (`0c0370c` 에서 땄다)

```text
   Functional Design      착수 2026-09-25T11:44:43Z.  앞 두 유닛이 넘긴 일을 받는다 — effect 에 따라 거두기 ·
                          두 예산 · discover 의 상한 (contract-grammar) · exited 보내기와 result 새 칸 여섯
                          (step-phase).  계획과 물음 아홉 2026-09-25T11:55:31Z — construction/plans/finalize-functional-design-plan.md.
                          답 2026-09-25T12:26:32Z — 아홉 모두 A (「답했다」).  되물음 없음.
                          산출물 셋 2026-09-25T12:26:32Z — construction/finalize/functional-design/.
                          승인 2026-09-25T12:53:36Z (「승인. nfr 건너뛴다.」)
   NFR Requirements       건너뛴다 — 사용자 결정 2026-09-25T12:53:36Z.  실행 계획은 「한다 (최소)」였다.
                          유닛 정의가 맡긴 것 중 조각 1 의 임대 창과 N3 (업로드 예산과 요청 · 흘려 보내기의
                          상한)은 FD 답 9 · 8 이 닫았다.  남는 값 둘 (30초에 훑는 항목 수 · helper 여유 5초)은
                          Code Generation 계획이 측정 자리로 받는다
   NFR Design             건너뛴다 (위와 같은 결정)
   Code Generation        Part 1 착수 2026-09-25T12:53:36Z.  계획 — 단계 열일곱 2026-09-25T13:01:31Z
                          construction/plans/finalize-code-generation-plan.md
                          NFR 이 맡았던 값 둘은 계획 3절 · FD 에 없던 자리 열하나 (계획 4절).  승인 2026-09-25T13:27:18Z (「승인」).
                          Part 2 착수 2026-09-25T13:27:18Z.  완료 2026-09-25T14:06:03Z — 단계 열일곱 중 열여섯과 반 체크
                          (커밋은 승인 뒤) · 코드 검사 초록 (통과 2,025 · 실패 0 · 스킵 0 · 패키지 전부 80% 이상 ·
                          internal/enode 81.5% -> 82.3% · 크로스 빌드 셋 · 린트 39 건 기준선과 같음) · 조각 0 초록 (라우트 19) ·
                          조각 1 초록 (300만 파일과 빈 워크스페이스 모두 finalized_at - exited_at 1 ms 아래 · stat 3) ·
                          조각 3 초록 (go test).  조각 2 는 사람의 조각 — slice-2.sh 준비.  측정 — 훑기 초당 약 304,000 방문 ·
                          300만 파일에서 방문 상한에 27.1 초 · 마감 뒤 돌아오기 1 ms 아래.  조각 1 은 이 브랜치의
                          스크래치 Mediator (127.0.0.1:18080) 로 돌렸다 — 개발용 :8080 은 2026-09-17 빌드.
                          요약은 construction/finalize/code/code-summary.md.  승인 2026-09-26T00:45:54Z (「승인」)
   병합                   조각 2 초록 — 에이전트가 돌리고 (「해봐」) 사용자가 판정 2026-09-26T03:33:12Z.  스크립트를 고쳐 다시 돌렸다
                          (369417e · 256 MiB 파일 하나가 blob 상한 10 MiB 에 413).  조각 1 · 2 · 3 초록.
                          PR #63 (CI 초록 · 린트 39 건 main 과 같음 · 커버리지 86.2%) 로 main 에 병합 (3f98c8f · 2026-09-26T03:46:10Z)
```

### U4 `trash` — 브랜치 `unit/trash` (`3f98c8f` 에서 땄다)

```text
   Functional Design      착수 2026-09-26T03:46:31Z.  finalize 가 넘긴 일을 받는다 — 닫기를 rename 으로 바꾸는 커밋에서
                          닫기를 Finalize 예산 안으로 (afterExit 의 session.Close 에 Finalize ctx) ·
                          Keep.Upper 가 "" 면 runRoot 를 trash 로 · finalized_at 의 뜻은 그대로.
                          계획과 물음 열 2026-09-26T03:51:52Z — construction/plans/trash-functional-design-plan.md.
                          답 2026-09-26T12:35:51Z — 열 모두 A (「권장대로」).  되물음 없음.
                          산출물 셋 2026-09-26T12:40:16Z — construction/trash/functional-design/.
                          승인 2026-09-26T14:24:01Z (NFR 을 건너뛰는 답으로 받았다 · 삭제자는 단계가 돌아도 지운다 — idle IO 는 BFQ 에서만)
   NFR Requirements       건너뛴다 — 사용자 결정 2026-09-26T14:24:01Z.  보안 · 성능은 FD 규칙 5절 · 1 · 2절이 닫았다.
                          N1 (한 번에 지우는 양과 속도)은 Code Generation 계획이 측정으로 받는다
   NFR Design             건너뛴다 (위와 같은 결정)
   Code Generation        Part 1 착수 2026-09-26T14:24:01Z.  계획 — 단계 열여덟 2026-09-26T14:26:50Z
                          construction/plans/trash-code-generation-plan.md
                          N1 은 계획 3절 · FD 에 없던 자리 여덟 (계획 4절).  SunnyVM 이 꺼져 있다 — 조각 4 는 켜진 뒤.  승인 2026-09-26T14:31:04Z (「승인한다.」).
                          Part 2 착수 2026-09-26T14:31:04Z.  완료 2026-09-26T15:08:14Z — 단계 열여덟 중 열일곱과 반 체크 (커밋은 승인 뒤) ·
                          코드 검사 초록 (통과 2,088 · 실패 0 · 스킵 0 · 스물한 패키지 전부 80% 이상 · 새 internal/scratch 93.5% ·
                          internal/enode 82.7% -> 83.6% · cmd/enode 83.0% -> 80.4% · 크로스 빌드 셋 · 린트 39 -> 38) ·
                          조각 0 초록 (라우트 19).  조각 4 는 사람의 조각 — slice-4.sh 준비 · 보류.  SunnyVM 은 다시 닿는다 (2026-09-26T15:08:14Z).
                          측정 N1 — BenchmarkRemove 150,101 항목에 3.14 초 (N100 · ext4 · mq-deadline).
                          요약은 construction/trash/code/code-summary.md.
                          미뤄 둔 실측 (2026-09-26T15:23:23Z · 사용자 지시) — 조각 4 를 에이전트가 SunnyVM 에서 돌렸다 (스크래치 Mediator :18080 ·
                          조각 전용 노드).  ① 208 ms · 43 ms · ② 9.67 GB 를 보고 뒤 1.09 초에 · ④ 남은 작업 폴더를 재시작 때 거둠 ·
                          ⑤ draining 과 arch 키 · ⑥ Open 실패와 smoke.  integration 시험 초록.  finalize 가 넘긴 둘도 닫음
                          (훑기 초당 약 811,700 · helper 여유 80 ~ 115 µs).  조각 4 초록 (사용자 판정) · 승인 2026-09-26T15:24:25Z
                          (「초록 · 승인 · PR 올림」).  한 커밋 (d5a258c) · PR #64
   병합                   PR #64 (CI 초록 · 린트 38 건 · 커버리지 하한 통과) 로 main 에 병합 (b44b46e · 2026-09-26T15:34:52Z).
                          조각 전용 자리 (SunnyVM ~/slice4 · ~/enode-trash-slice · slice4.yaml · DB enode_slice4 · 스크래치 Mediator) 는 지웠다
```

### U5 `merge-rules` — 브랜치 `unit/merge-rules` (`b44b46e` 에서 땄다)

```text
   Functional Design      착수 2026-09-26T15:34:52Z.  맡는 것 — FR-7 의 합치기 규칙 · 조각 7 의 기계 부분 (가짜 트리 재개 시험).
                          실측 넷 (SunnyVM 커널 7.0 · 이 기계) — 시제품 merge.py 가 종류가 바뀐 항목을 lower 쪽을 trash 로 옮긴 뒤
                          대체했다 · user namespace 마운트는 커널이 userxattr · redirect_dir=nofollow 를 스스로 붙이고 metacopy 가
                          꺼져 있어 제품 upper 의 표시가 ADR-077 §4 의 전제 그대로다 · 복사해 올린 항목에 user.overlay.origin 이
                          붙는다 · 형제 helper 의 overlay 마운트는 호스트 mountinfo 에 안 보이고 /proc/<pid>/mountinfo 에서 보인다.
                          계획과 물음 아홉 2026-09-26T15:41:31Z — construction/plans/merge-rules-functional-design-plan.md.
                          다시 검토 2026-09-26T15:55:03Z — 물음 1 · 5 · 6 · 8 을 고쳤다 (Application Design 의 결정 · 봉인 · trash 유닛의
                          넘김과 맞춤 · 측정 둘 더 — upper 의 xattr whiteout 은 커널이 반쪽만 읽는다 · origin 이 lower 에 남아도 정상).
                          답 2026-09-26T16:02:51Z — 아홉 모두 A (「권장대로」).  산출물 셋 2026-09-26T16:09:11Z — construction/merge-rules/
                          functional-design/.  답 8 에 따라 회차 문서 셋 (unit-of-work.md 5절 · components.md 2.2 · component-methods.md 2절)
                          의 xattr whiteout 줄을 고쳤다.  재개 때 디렉터리 mtime 을 맞추려고 upper 디렉터리에 user.enode.merge-mtime 을
                          적어 두기로 했다 (시제품은 안 했다).  승인 2026-09-26T16:11:17Z (「nfr 건너뛰고 다음」)
   NFR                    건너뜀 (사용자 결정 2026-09-26T16:11:17Z).  유닛 정의가 NFR 에 둔 것 — 보안 확인 (FD 규칙 3절 · 7절) · 하루치 규모의
                          Preflight 와 Apply 시간 — 을 Code Generation 계획이 받는다 (trash 유닛과 같은 방식)
   Code Generation        착수 2026-09-26T16:11:17Z
```

## Stage Progress

### INCEPTION PHASE
- [x] Workspace Detection — 2026-09-23T13:49:57Z
- [x] Reverse Engineering — 전면 갱신 2026-09-23T14:03:35Z · 승인 2026-09-23T14:12:44Z
- [x] Requirements Analysis — 착수 2026-09-23T14:20:56Z. 답 14:35:56Z · 재질문 답 14:41:53Z. `requirements.md` 완료 14:50:03Z · 승인 2026-09-23T14:56:31Z
- [x] User Stories — 판정 · 계획 · 질문 다섯 2026-09-23T15:00:55Z. 답(권장대로) 23:42:13Z. 페르소나 넷 · 스토리 열아홉 · 완료 조건 열. 승인 2026-09-23T23:54:49Z
- [x] Workflow Planning — 계획 2026-09-23T23:57:36Z · 승인 2026-09-24T01:14:24Z
- [x] Application Design — 계획 · 질문 일곱 2026-09-24T01:23:48Z. 채팅 논의로 일곱 다 A (Q7 06:33:21Z). 산출물 다섯 2026-09-24T06:39:39Z · 승인 2026-09-24T06:42:48Z
- [x] Units Generation — 착수 2026-09-24T06:42:48Z. 계획 · 질문 넷 2026-09-24T06:50:29Z. 채팅 논의로 일곱이 닫힘 2026-09-24T09:50:00Z (Q1 A · Q2 A · Q3 A 순연 · Q4 B · Q7 A). 계획 승인 2026-09-24T12:14:10Z. 산출물 넷 2026-09-24T12:22:43Z · 승인 2026-09-24T12:31:50Z

### CONSTRUCTION PHASE
- [ ] Functional Design — EXECUTE (유닛마다). U1 contract-grammar 완료 2026-09-25T05:03:44Z · U2 step-phase 완료 2026-09-25T08:52:58Z · U3 finalize 완료 2026-09-25T12:53:36Z · U4 trash 완료 2026-09-26T14:24:01Z · U5 merge-rules 완료 2026-09-26T16:11:17Z — 위 「Construction」 절
- [ ] NFR Requirements — EXECUTE (유닛마다 · 최소). U1 ~ U5 는 건너뜀 (U1 유닛 정의 · U2 ~ U5 사용자 결정)
- [ ] NFR Design — EXECUTE (유닛마다 · 최소). U1 ~ U5 는 건너뜀
- [ ] Infrastructure Design — SKIP
- [ ] Code Generation — EXECUTE (유닛마다). U1 contract-grammar 완료 2026-09-25T05:45:28Z · U2 step-phase 완료 2026-09-25T11:23:52Z · U3 finalize 완료 2026-09-26T00:45:54Z · U4 trash 완료 2026-09-26T15:24:25Z
- [ ] Build and Test — EXECUTE (조각 0 ~ 12)

### OPERATIONS PHASE
- [ ] Operations — PLACEHOLDER
