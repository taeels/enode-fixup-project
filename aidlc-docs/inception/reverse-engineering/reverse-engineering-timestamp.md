# Reverse Engineering Metadata

**Analysis Date**: 2026-09-23T14:03:35Z
**Analyzer**: AI-DLC v1.0.1 — `v4-run-finalize-bake` 회차의 Reverse Engineering 단계
**Workspace**: /home/sunny/enode-fixup-v4 (git worktree)
**Baseline commit**: `195a5d0` (`origin/main` `073f5f1` 위에 `unit/runtime-environment-profile` 을 합친 나무)
**enode-design pin**: `369270a`
**Total Files Analyzed**: 비테스트 Go 132 파일 31,551 줄 · 전체 Go 263 파일 ·
브라우저 자산 `internal/api/ui/static/**` · `internal/transcriptui/card.mjs` · `enode-design` 정본

## Artifacts Generated
- [x] business-overview.md
- [x] architecture.md
- [x] code-structure.md
- [x] api-documentation.md
- [x] component-inventory.md
- [x] technology-stack.md
- [x] dependencies.md
- [x] code-quality-assessment.md

---

## 왜 돌았나

`workspace-detection.md` Step 3 의 분기다. 산출물이 있으나 **이 회차가 딛는 경로에서
낡았다.** 사용자에게 안 물었다 — 규칙이 정한다.

```text
   기준선 5716714 -> 195a5d0     Go 파일 131 변경 · 16,262 줄 추가 · 632 줄 삭제
   패키지                         18 -> 21
   이 팩의 접점 파일               산출물에 나온 횟수
     internal/environment/*       0
     runc_overlay_linux.go        0
     StepRuntime                  0
   라우트                         26 -> 27
```

`requirements/finalize-bake/constraints.md` 3절의 접점 표가
`internal/enode/runc_overlay_linux.go` 와 `internal/environment/check.go` 를 만진다.
없는 것을 딛고 Requirements 를 쓰면 그 문서가 코드를 안 보고 쓴 것이 된다.

## 범위 — 전면이다

부분 갱신은 안 잰 자리를 두 겹으로 쌓는다 (앞 회차의 판단을 잇는다). 여덟 문서를
전부 다시 썼다. 두 갈래의 변경이 한꺼번에 들어왔다 — 트랜스크립트 회차의 유닛
여덟(`main`)과 실행 환경 구현(`unit/runtime-environment-profile`, `main` 에 아직 없다).

## 방법

```text
   읽었다      이 회차가 딛는 경로를 전문으로 — internal/enode 의 runtime · runc_overlay_linux ·
               environment · claim(execute · runAgentStep · Result) · detect(hasRoom · 광고 키) ·
               overlay_linux 머리 · internal/environment 의 profile · check 머리 · manifest ·
               record · apply 의 단계 · cmd/enode 의 main 과 environment ·
               internal/store 의 ReportStep · reap 의 진행 쓸기 · schema.sql 의 steps
   훑었다      기준선 뒤 커밋 서른의 메시지 · 변경 파일의 diff · 나머지 패키지의 머리 주석과
               최상위 선언 · internal/api 의 log.go · internal/record 의 progress.go ·
               internal/transcript · internal/panel 의 변경 · 경계 검사 표
   세었다      go list ./... (21) · mux 등록 (27) · 제어판 등록 (11) ·
               go list -f Imports 로 의존 그래프 · go list -deps ./cmd/mediator ·
               find 로 파일 수와 줄 수 · func Test 선언 수
   돌렸다      go build ./... (exit 0) · go vet ./... (exit 0)
               go test ./... -count=1 -coverpkg=./... -coverprofile -json
               (exit 0 · 실패 0 · 스킵 0 · 85.3% · 스무 패키지 전부 80% 이상 · 1분 20초)
               ENODE_TEST_DATABASE_URL 을 세우고 .github/ci-stubs 를 PATH 에 두었다
   못 잰 것    golangci-lint (이 기계에 없다) · 크로스 빌드 · Windows 플랫폼 파일 ·
               integration 태그 시험 (rootfs 와 환경변수 셋이 필요하다)
```

측정이 작업 트리를 한 번 더럽혔다 — 테스트가 `cmd/enodectl/probe.lock` 을 제자리에서
바꾼다 (앞 판이 부채로 적은 그대로다). `git checkout` 으로 되돌렸다.

## 검증

- `enode-design/scripts/emphasis-check.py` 를 여덟 문서와 이 파일에 돌려 전부 통과 (exit 0)
- U+2605 없음 · 코드펜스 짝 맞음
- Mermaid 넷 (architecture 2 · business-overview 1 · dependencies 1) — 노드 ID 가
  영숫자와 밑줄뿐이고 라벨이 따옴표 안에 있다
- 라우트 27 이 `internal/api/*.go` 의 `mux.HandleFunc` + `mux.Handle(` 원본과 일치
- 의존 그래프 간선이 `go list -f '{{.Imports}}'` 출력과 일치

---

## 이전 판

### 2026-09-15T04:10:00Z — 전면 (대체됨)

`v3-run-transcript` 회차. 기준선 `5716714`. 비테스트 97 파일 · 전체 198. 라우트 26.
패키지 열여덟. 커밋 `536c396`. 완료 시각이 승인 시각보다 뒤로 적혀 측정이 아니라는
기록이 그 회차의 상태 파일에 있다.

### 2026-09-11T13:13:54Z — 부분 (대체됨)

네 경로만. 두 문서만 고쳤다.

### 2026-09-08T07:17:58Z — 전면 (대체됨)

비테스트 75 파일 · 라우트 15 · 패키지 열둘. 커밋 `1fe2145`.

---

## 옛 판이 「없다」로 적었으나 지금 있는 것

```text
   GET /v1/runs/{run}/steps/{seq}/log   봉인 전은 진행 파일, 봉인 뒤는 logs/
   진행 파일                             <Root>/progress/run-<id>/.  봉인이 걷고 회수기가 쓴다
   도는 동안 올리는 경로                  PUT log?progress=1.  2초 또는 64 KiB
   StepRuntime · runc-overlay            ADR-073.  단계마다 userns · overlay · runc
   실행 환경 profile · env check · apply  ADR-073.  internal/environment
   광고 키 셋                             machine · arch.<이름> · overlay
   internal/transcript · transcriptui     파서 한 벌 · 카드 렌더러 한 장
```

## 옛 판이 「있다」로 적었으나 지금 다른 것

```text
   에이전트 단계의 링 tee 없음     되살아났다
   logs/ 는 허용목록 선별          크기 절단이다 (ADR-071).  말과 도구 호출은 전문
   Decode 가 배치라 사건이 안 흐름   흘리기(emit.go)와 판정(Decode)이 갈렸다
   AppendLog 가 이번 호출의 바이트   파일 총 길이를 낸다.  PUT log 가 200 과 헤더로 싣는다
   계약의 workspace.rev            없다 (ADR-072).  노드가 checkout 을 안 한다
   제어판이 봉인 tar 를 푼다        GET .../log 로 받는다
   두 나무가 contract 만 공유       internal/environment 도 공유한다
```
