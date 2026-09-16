# Reverse Engineering Metadata

**Analysis Date**: 2026-09-15T04:10:00Z
**Analyzer**: AI-DLC v1.0.1 — `v3-run-transcript` 회차의 Reverse Engineering 단계
**Workspace**: /home/sunny/enode-fixup-project
**Baseline commit**: `5716714` (`v3-run-transcript` 가 `main` 을 합친 커밋)
**Total Files Analyzed**: 비테스트 Go 97 파일 24,534 줄 · 전체 Go 198 파일 ·
브라우저 자산 `internal/api/ui/static/**` · `enode-design` 정본

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

`workspace-detection.md` Step 3 의 분기다. 산출물이 있으나 **이 회차가 읽는
경로에서 낡았다.** 사용자에게 안 물었다 — 규칙이 정한다.

```text
   internal/panel     패키지 전체가 신규.  옛 산출물에 이름이 한 번(dependencies) 뿐
   internal/api/ui    패키지 전체가 신규.  옛 산출물에 이름이 0 번
   internal/api       라우트 15 -> 26.  옛 api-documentation.md 는 열둘을 적었다
```

`requirements/transcript/` 의 3.3 이 `internal/panel` 을, 3.6 이 `internal/api/ui` 를
만진다. 없는 것을 딛고 Requirements 를 쓰면 그 문서가 코드를 안 보고 쓴 것이 된다.

## 범위 — 전면이다

부분 갱신을 한 번 더 하면 안 잰 자리가 두 겹으로 쌓인다 (2026-09-11 판이 이미
「안 잰 자리의 알려진 낡음」을 관측값으로만 적고 본문을 안 고쳤다). 여덟 문서를
전부 다시 썼다.

## 방법

```text
   읽었다      이 회차가 딛는 경로를 전문으로 — internal/panel 다섯 · internal/proc 셋 ·
               internal/api/ui/ui.go · internal/api 의 nodes·runs·ratelimit ·
               internal/enode 의 transcript·claude·runner·harness ·
               internal/record/record.go · api.go 의 라우팅과 putLog·putBlob·getRecord
   훑었다      나머지 패키지의 파일 머리 주석과 최상위 선언
   세었다      go list ./... (18) · mux 등록 (26) · go list -f Imports 로 의존 그래프 ·
               find 로 파일 수와 줄 수
   돌렸다      go build ./... (exit 0) · go vet ./... (exit 0)
               go test ./... -count=1 -coverpkg=./... (exit 0 · 스킵 0 · 87.4%)
               ENODE_TEST_DATABASE_URL 을 세우고 .github/ci-stubs 를 PATH 에 두었다
   못 잰 것    golangci-lint (이 기계에 없다) · 크로스 빌드 · Windows 플랫폼 파일 셋
```

## 검증

- `enode-design/scripts/emphasis-check.py` 를 여덟 문서에 돌려 전부 통과 (exit 0)
- U+2605 없음 · 장식 문자 없음 · 코드펜스 짝 맞음
- Mermaid 넷 (architecture 2 · business-overview 1 · dependencies 1) 문법 유효
- 라우트 26 이 `internal/api/*.go` 의 `mux.HandleFunc` + `mux.Handle(` 원본과 일치
- 의존 그래프가 `go list` 출력과 글자까지 일치

---

## 이전 판

### 2026-09-08T07:17:58Z — 전면 (대체됨)

비테스트 75 파일 · 테스트 포함 143. 라우트 15. 패키지 열둘. 워크플로로 돌렸다
(리더 7 · 라이터 6). 커밋 `1fe2145`.

### 2026-09-11T13:13:54Z — 부분 (대체됨)

네 경로만 — `internal/enode` · `internal/contract` · `cmd/iapadapter` ·
`cmd/runctl`. `harness-components` 회차의 질문 1 의 답이 B 였다. 두 문서만 고쳤다
(`code-structure.md` · `component-inventory.md`).

**그 부분 갱신이 질문으로 갈린 것을 그 회차의 검증이 규칙 위반으로 셌다** —
Workspace Detection 이 규칙 분기 대신 사용자에게 물었다. 이번 회차는 안 물었다.

---

## 옛 판이 「없다」로 적었으나 지금 있는 것

```text
   GET /v1/nodes              ADR-065.  관측된 함대 스냅샷
   GET /v1/runs               ADR-065.  좁히기 넷 · 최신이 앞
   GET /ui/ 정적 서빙          화면 번들 넷
   202 QUEUED · WakeQueued    ADR-064.  internal/store/queue.go
   소유자 drain 정책           ADR-063.  internal/enode/policy.go + 광고 + 제어판
   상태 파일                   ADR-068.  internal/enode/status.go
   트랜스크립트 링             internal/enode/transcript.go
   노드 제어판                 internal/panel + enodectl serve + enode panel
   requires 필드               ADR-069.  GET /v1/runs/{id} 응답
```

## 옛 판이 「있다」로 적었으나 지금 다른 것

```text
   하네스 stdout 의 링 tee     껐다 (짝 팩 decisions 6절 ⑲).  명령 단계만 흐른다
   logs/ 는 원문 그대로        허용목록으로 걸러진다 (⑱).  ADR-005 와 어긋나는 자리다
   --output-format json        stream-json --verbose 다 (⑮)
   라우트가 전부 s.auth        데모 모드는 읽기 셋이 무인증 + 한도다.  ADR-065 §2 를 벗어난다
```
