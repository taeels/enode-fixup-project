# AI-DLC State Tracking — v4-run-finalize-bake

## Project Information
- **Project Type**: Brownfield
- **Start Date**: 2026-09-23T13:49:57Z
- **Current Stage**: INCEPTION - User Stories (착수 — Part 1 Planning)
- **AI-DLC Version**: 1.0.1 (`.aidlc/aidlc-rules/`)
- **Run Branch**: `v4-run-finalize-bake` (Inception 산출물이 여기 직렬로 쌓인다. CONVENTIONS.md 3.1)
- **문서 루트**: `aidlc-docs/v4-run-finalize-bake/` (CLAUDE.md 의 회차별 layering)
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

## Stage Progress

### INCEPTION PHASE
- [x] Workspace Detection — 2026-09-23T13:49:57Z
- [x] Reverse Engineering — 전면 갱신 2026-09-23T14:03:35Z · 승인 2026-09-23T14:12:44Z
- [x] Requirements Analysis — 착수 2026-09-23T14:20:56Z. 답 14:35:56Z · 재질문 답 14:41:53Z. `requirements.md` 완료 14:50:03Z · 승인 2026-09-23T14:56:31Z
- [ ] User Stories — 돈다 (requirements.md 9절 — 새 사용자 기능 · 워크플로 변경 · 여러 페르소나)
- [ ] Workflow Planning
- [ ] Application Design
- [ ] Units Generation
