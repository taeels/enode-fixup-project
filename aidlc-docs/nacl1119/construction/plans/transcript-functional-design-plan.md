# transcript — Functional Design 계획 (Part 1)

AI-DLC Construction · **transcript 유닛**(W4 · CP6 · 담당 nacl1119)의 Functional
Design 이다. 값은 `decisions.md` §6.2/§6.3/§6.5 가 이미 닫았으므로 **막는 결정이
없다** — 계획을 내고 산출물로 이어 진행한다.

정본 — 유닛 정의 `unit-of-work.md` §6 · `decisions.md` §6 전체(6.2 값 셋 · 6.3 링
파일 · 6.4 안 하는 것 · 6.5 지난 작업) · enode-features 3.1.3 · scene-gates CP6 ·
`design/enode-ux.pen` S4(자리만)와 S3 의 카드.

---

## 0. 이 유닛이 지는 것 (한눈)

```text
   신규 파일   internal/enode/transcript.go   고정 크기 링 파일 (Ring) — 쓰기·읽기·비우기
   만지는 것   internal/enode/runner.go       하네스 stdout 을 링에 tee (io.MultiWriter 한 겹)
              internal/enode/claim.go        명령 단계 stdout/stderr 를 링에 tee
              internal/enode/(worker 배선)    단계 시작 때 Reset · Job 에 링 writer 전달
              internal/panel                 트랜스크립트 카드(GET /api/transcript) ·
                                             지난 작업(GET /api/runs · /api/record)
   딛는 표면   runctl.Client.Runs(원문) · Record(tar)  — obs 병합됨
   지는 기능  enode-features 3.1.3.  재는 게이트 CP6 (장면 밖 · CP4 뒤 아무 때나)
   의존       panel(카드가 사는 화면 · main 에 병합됨) · obs(GET /v1/runs · main)
   Mediator   변경 0 (decisions §6.4).  cmd/mediator · internal/api 안 건드린다
```

---

## 1. 착수 전에 실측한 것 (기존 코드)

```text
   확인한 사실                                                        자리
   ─────────────────────────────────────────────────────────────     ────────────────────────
   하네스 출력이 도는 동안 어디에도 없다 — bytes.Buffer 로 통째로 받아   internal/enode/runner.go:94~96,106
     cmd.Run() 이 끝나야 Decode. 데몬도 끝 전엔 못 본다 (decisions §6.1)
   명령 단계도 같은 패턴 — var buf bytes.Buffer; cmd.Stdout/Stderr=&buf  internal/enode/claim.go:596~600
   panel 이 이미 정책·상태 파일을 <stem>.*.yaml 로 읽고 쓴다            internal/enode/policy.go·status.go
     — 링 파일도 같은 자리·같은 권한(0600 · 상속 ACL)
   runctl.Client.Runs(RunsQuery) 가 원문 JSON 을 낸다                   internal/runctl/client.go:342
   runctl.Client.Record(runID, w) 가 tar 를 그대로 흘린다              internal/runctl/client.go:217
   GET /v1/runs 응답의 assigned 가 노드를 싣는다([{as,nodes:[{node,label}]}])  decisions §6.5
     — 제어판이 자기 node_id 로 거른다. node= 필터 신설 없음
   panel 은 지금 데몬 로그(<name>.log)만 보인다 — 트랜스크립트는 0 줄   internal/panel/handlers.go handleLogs
   archive/tar 는 표준 — 새 의존 0                                     (decisions §6.5 「고른 이유」)
```

---

## 2. 물음 — 없음

`decisions.md` §6 이 값을 다 닫았다(크기 512 KiB · 갱신 1초 · 링 머리/몸통/감김/
비우기 · 지난 작업은 GET /v1/runs + record tar · Mediator 무변경). 진행자 결정이
필요한 미결이 없다. panel 의 ADR-068 같은 블로킹이 이 유닛엔 없다.

---

## 3. Functional Design 실행 계획

### 3.1 domain-entities.md — 링 파일과 카드가 쓰는 것

- [x] `Ring` 파일 형식 — 머리(magic · 판 · 용량 512 KiB · 총 쓴 바이트 · 세대) · 몸통(용량 바이트)
- [x] `TranscriptPath(configPath)` = `<dir>/<stem>.transcript` (PolicyPath/StatusPath 와 같은 규칙)
- [x] `Job.Transcript io.Writer` — runHarness 에 tee 대상을 넘기는 자리 (nil 이면 안 흘린다)
- [x] 트랜스크립트 카드 view — 링에서 읽은 최근 바이트(순서대로) · 세대(비움 감지)
- [x] 지난 작업 view — GET /v1/runs 를 자기 node_id(assigned)로 거른 목록(run_id·state·
      ended_at·verdict.checks) · 누르면 record tar 의 logs/NN-*.log

### 3.2 business-logic-model.md — 흐름

- [x] 쓰기 — runner.go 가 cmd.Stdout 을 io.MultiWriter(&stdout, ring) 로 감싼다(하네스 원문 stdout · Q2).
      claim.go 도 buf 를 io.MultiWriter(&buf, ring) 로. Decode/로그는 그대로 &stdout/&buf 를 읽는다
- [x] 링 쓰기 — WriteAt((총량 % 용량))에 쓰고 끝에서 감긴다. 머리의 총량을 갱신. Truncate·rename·삭제 안 함
- [x] 비우기 — 단계가 시작할 때 Worker 가 Ring.Reset()(총량 0 · 세대+1). 끝날 때가 아니다
- [x] 읽기(panel) — 머리 · 몸통 · 머리 다시 읽어 많이 움직였으면 한 번 더(찢긴 읽기 완화)
- [x] 카드 폴링 — 화면 열려 있을 때만 1초. Mediator 폴링(5초)과 별개 타이머
- [x] 지난 작업 — GET /api/runs -> runctl.Runs 원문 -> assigned 로 거른 목록.
      GET /api/record?run=<id> -> runctl.Record tar -> logs/NN-*.log 추출 + verdict.checks
- [x] 흐름 ASCII 다이어그램

### 3.3 business-rules.md — 규칙과 완료 조건

- [x] 윈도우가 설계를 정했다 — Truncate·rename·삭제 안 하는 모양이라 빌드 태그 쌍이 하나도 안 는다(decisions §6.3)
- [x] 상한을 넘겨도 안 깨지고 오래된 줄부터 밀린다(링). 다음 단계 첫 글자에 갈린다(Reset)
- [x] 권한 — 유닉스 0600 · 윈도우 상속 ACL (정책·상태 파일과 같다)
- [x] Mediator 변경 0 · 하네스 계약(--output-format) 무변경 · 새 의존 0(archive/tar 표준)
- [x] 트랜스크립트 카드는 데몬 로그 카드와 다른 물건 — 화면에서 둘이 갈려 보인다(CP6)
- [x] S4 는 자리만 · 살아나는 것은 S3 의 카드 (decisions §6.1 · scene-gates CP6)
- [x] 완료 조건 CP6 — 도는 것(단계 끝 전 흐름 · 상한 넘겨도 앞부터 밀림 · 다음 단계 첫 글자에 갈림) ·
      지난 것(결과 verdict.checks + 봉인 트랜스크립트) · 커버리지 80%(internal/enode)

### 3.4 검토

- [x] CONVENTIONS 1·2 · content-validation · security-baseline 요약
- [x] 2지 선택 완료 메시지 — 결정 없으면 자동으로 다음 단계(진행자 위임)

---

## 4. 낼 파일

```text
   aidlc-docs/nacl1119/construction/transcript/functional-design/
     domain-entities.md        링 파일 형식 · 카드/지난작업 view
     business-logic-model.md   tee · 링 쓰기/읽기/비우기 · 폴링 · 지난 작업
     business-rules.md         윈도우 제약 · 권한 · Mediator 무변경 · CP6 완료 조건
```

---

## 5. 이 유닛이 안 하는 것 (decisions §6.4)

```text
   stream-json 전환 · SSE/WebSocket · Mediator 중계 · 트랜스크립트 구독 · node= 필터 신설 ·
   cmd/mediator·internal/api 변경 · 장면(dhseo) 변경
```

---

## 6. 지금 상태

- 브랜치 `unit/transcript` (main 77f5a83 에서 딴 것 · upstream 없음 · 첫 push 는 git push -u)
- 의존 닫힘 — panel(PR #9·#10) · obs(#4) main 에 있음
- 막는 결정 없음. 이 계획 뒤 FD 산출물 -> NFR -> 코드로 진행
