# AI-DLC Audit Log — shin-son (손신)

Construction 담당 shin-son 의 감사 로그다. 유닛 queue · drain · panel · transcript 의
사용자 입력과 결정을 시간순으로 기록한다. 사용자 원문은 요약하지 않고 그대로 싣는다.
타임스탬프는 `date -u +%Y-%m-%dT%H:%M:%SZ` (ISO 8601 UTC).

---

## Session Continuity — 착수 요청
**Timestamp**: 2026-09-08T13:53:27Z
**User Input**: "main 최신으로 당겨, 그리고 branch 새로 따서, 이제 내가 할당받은 (shin-son) 부분 작업 시작할거야."
**AI Response**: `main` 을 origin 에서 fast-forward 로 당겼다(팁 `c010841` CONSTRUCTION 착수 커밋). 유닛 브랜치 `unit/queue` 를 `main` 에서 땄다(CONVENTIONS 3.1 · 첫 유닛이 queue 라서). 담당 상태 `aidlc-docs/shin-son/aidlc-state.md` · 배정 `construction-roster.md` · CLAUDE.md 의 담당별 문서 루트 규약을 읽고, AI-DLC 규칙(core-workflow · common 다섯 · construction/functional-design · 확장 opt-in 셋)을 로드했다. 켜진 확장 security-baseline 은 `decisions.md` §3 의 취급표를 따른다.
**Context**: CONSTRUCTION 착수. 담당 shin-son 의 첫 유닛 queue.

---

## Session Continuity — 맥락 로드 · 발견
**Timestamp**: 2026-09-08T13:53:27Z
**User Input**: (없음 — 위 요청의 후속)
**AI Response**: 읽은 것 — 유닛 정본 `unit-of-work.md` §2 queue · `unit-of-work-dependency.md`(파일 행렬 · 접점 넷) · `unit-of-work-story-map.md`(CP2) · 요구 팩(`enode-features.md` 3.2.1 · `decisions.md` §1·§2·§7 · `scene-gates.md` CP2 · `constraints.md`) · 정본(`ADR-064` · `INVARIANTS` §1·§2 · `mediator-api.md` §1·§2 · `ADR-063` §2.1·§4·§6) · 코드(`internal/store` 의 CreateRun·CreateRejectedRun·BusyNodes·Cancel·SettleIfDone·Reap·FailRestarted·ReportStep·applyRelease·UpsertAdvert · `internal/api/api.go` submit·postResult·postCancel·postNodes · `cmd/mediator/main.go` · `schema.sql`). 정본은 서브모듈 `enode-design` 이 비어 있어 `git submodule update --init` 으로 채웠다(코드 변경 아님).

발견 셋.
1. **선행 obs(taeels · W0)가 아직 main 에 없다** — `internal/api/nodes.go`·`runs.go` 없음, 원격에 obs 브랜치 없음. Functional Design 은 obs 코드를 안 딛으므로 지금 하고, Code Generation 의 접점 파일(`internal/store/store.go` · `internal/api/api.go` · `cmd/mediator/main.go`)은 obs 병합 뒤 rebase 해서 만진다.
2. **`nodes.draining` 열이 아직 없다** — 유닛 정본은 「열은 이미 있음」이라 적었으나 `schema.sql` 과 Go 코드 어디에도 없다. 주인을 정해야 한다(계획 파일 질문 1).
3. **로컬에 Postgres · docker 가 없다** — `internal/store` 의 DB 테스트가 하드 실패하고 `internal/api` 는 건너뛴다(CP0 주석 그대로). Code Generation 전에 `scripts/testdb.sh` 를 돌릴 환경이 필요하다.

`runctl` 은 2xx 를 성공으로 보므로(`internal/runctl/client.go:70`) 202 에 손댈 것이 없다 — ADR-064 §2 그대로. 파일 행렬 밖을 안 만진다.
**Context**: CONSTRUCTION - queue - Functional Design 준비.

---

## Functional Design — queue · 계획 · 질문 셋
**Timestamp**: 2026-09-08T13:53:27Z
**User Input**: (없음 — 계획 파일을 낸다)
**AI Response**: `aidlc-docs/shin-son/construction/plans/queue-functional-design-plan.md` 를 냈다. 산출물 계획(체크박스) · 설계 초안 · 물음 없이 정한 것과 근거 · 계약에 걸리는 질문 셋([Answer]: 태그). 질문을 셋으로 제한한 근거는 이 저장소의 착수 지침(`v1-run-dhseo/audit.md` 최초 항목 — 「계약(함수 서명 · 파일 경로 · 응답 모양)에 걸리는 것만, 단계마다 셋까지」)이다. 답을 받으면 FD 산출물 셋(business-logic-model · business-rules · domain-entities)을 `aidlc-docs/shin-son/construction/queue/functional-design/` 에 낸다.
**Context**: CONSTRUCTION - queue - Functional Design (Step 2~4 완료 · Step 5 답 대기).

---

## Session Continuity — 코드 최신화 (obs 병합 반영)
**Timestamp**: 2026-09-08T16:15:25Z
**User Input**: "미안, 지금 코드 다시 최신화 해줄래? obs 마무리 되었을거야."
**AI Response**: origin 을 당겼다 — `main` 이 `c010841` 에서 `0a159a4`(PR #4 `unit/obs` 병합)로 여덟 커밋 앞섰다. 미커밋 문서 셋을 stash 로 지키고 `main` 을 fast-forward 한 뒤 `unit/queue` 를 그 위로 rebase 하고 stash 를 되돌렸다(충돌 없음). `go build ./...` · `go vet` 통과.

같이 들어온 것 넷. (1) **재배정** `81b526c` — shin-son 은 queue · drain, panel · transcript 는 nacl1119 로. 담당 상태 파일이 그 판으로 바뀌었고 내 진행 절이 그 위에 얹혔다. (2) **obs 가 `nodes.draining` 열을 더했다** — 어휘 셋과 `store.DrainPolicy`, `UpsertAdvert` 가 이전 drain 값을 돌려준다(해제 지점용). 계획의 질문 1 이 닫혀 질문은 둘이 됐다. (3) `runs.submitter` · `Run.Submitter` · `runView.Requires`(ADR-069) — QUEUED 가 무엇을 기다리는지는 obs 의 상세가 낸다. (4) `docs/testdb-setup.md` — Docker 없이 Postgres 17 을 까는 안내. 이 기계에는 docker 도 brew 도 없어 Code Generation 전에 깔아야 한다.

계획 파일 §0 · §3.4(drain 해제 지점을 queue 가 postNodes 에서 이전 값으로 잡는다) · §3.6 줄 번호 · §5 를 고쳤다.
**Context**: CONSTRUCTION - queue - Functional Design (답 대기 유지).

---

## Functional Design — queue · 질문 답 (일부)
**Timestamp**: 2026-09-08T17:27:35Z
**User Input**: "Q1 다시 설명해/
Q2 A"
**AI Response**: Q2(QUEUED Run 의 steps 행 시점) 답 A — 승격 때 만든다 — 를 계획 파일에 적었다. Q1(CreateQueuedRun 서명)은 채팅에서 다시 풀어 설명하고 답을 기다린다.
**Context**: CONSTRUCTION - queue - Functional Design (Step 5 · 답 수집 중).

---

## Functional Design — queue · 답 수집 완료 · 산출물 생성
**Timestamp**: 2026-09-08T17:32:59Z
**User Input**: "A"
**AI Response**: Q1(CreateQueuedRun 서명) 답 A — store 가 자기 tx 를 열고 잠금·INSERT·첫 훑기를 진다 — 를 계획 파일에 적었다. 모호한 답 없음. FD 산출물 셋을 `aidlc-docs/shin-son/construction/queue/functional-design/` 에 냈다 — business-logic-model.md(제출 두 자리 · CreateQueuedRun · wakeQueuedIn/promoteIn · 지점 여섯+기동 표 · 예외 흐름 · 닿는 파일) · business-rules.md(상태 · 정산 보호 · 응답 · 순서·동시성 · 보안 확장 취급 · 안 하는 것) · domain-entities.md(QUEUED 행 모양 · 어휘 · Store 겉면 넷과 정본 대비 · applyRelease 반환 · 파일 행렬). 코드를 읽다 잡은 것 하나 — SettleIfDone 이 단계 0 인 Run 을 SUCCEEDED 로 닫을 수 있어 RUNNING 만 정산하는 보호를 규칙으로 뒀다. 이 유닛이 정본에서 벗어나는 것 하나 — CreateQueuedRun 겉면(근거는 domain-entities §3). 계획 체크박스 갱신. 완료 메시지(2-옵션)를 내고 승인을 기다린다.
**Context**: CONSTRUCTION - queue - Functional Design (Step 6·7 완료 · Step 8 승인 대기).

---

## Functional Design — queue · 검토 질문
**Timestamp**: 2026-09-08T17:34:35Z
**User Input**: "정본에서 벗어났다는건, U0 obs 설계에서 벗어난거야?"
**AI Response**: 아니다. 벗어난 것은 `unit-of-work.md` §2(Units Generation 산출물)의 queue 겉면 스케치 한 줄 — `CreateQueuedRun` 서명 — 이고 Q1 답 A 의 결과다. 설계 정본(`INVARIANTS` · `ADR-064` · `mediator-api`)과 obs 의 접점(`nodes.draining` · `DrainPolicy` · `UpsertAdvert` 반환 · `Run.Submitter` · `runView.Requires`)은 무변경. 완료 메시지의 「정본」이라는 말이 느슨했다고 정정했다.
**Context**: CONSTRUCTION - queue - Functional Design (승인 대기 중 검토 문답).

---

## Functional Design — queue · 승인
**Timestamp**: 2026-09-08T17:37:00Z
**User Input**: "승인할게."
**AI Response**: Functional Design 승인을 기록했다. 상태 파일의 queue FD 를 완료로 표시하고, 이 단계의 산출물(계획 · FD 셋 · 상태 · 감사)을 `unit/queue` 에 커밋한다(CONVENTIONS 3.3 「단계 승인마다」). 다음은 NFR Requirements — `construction/nfr-requirements.md` 를 읽고 실행 여부를 판단한다.
**Context**: CONSTRUCTION - queue - Functional Design 승인 (Step 9).

---

## NFR Requirements — queue · 계획 · 물음 셋
**Timestamp**: 2026-09-08T17:39:36Z
**User Input**: (없음 — FD 승인의 후속)
**AI Response**: NFR Requirements 를 실행한다 — 보안 확장이 켜져 있어 규칙 열다섯의 판정을 유닛마다 낸다(obs 선례와 같은 자리). 계획 `construction/plans/queue-nfr-requirements-plan.md` 를 냈다: 실측 표 · 닫힌 값 · 권장값(훑기 상한 없음 · 잠금은 막고 기다림 · 로그 규칙) · 물음 셋 — (1) QUEUED 부분 인덱스(schema.sql 접점 한 줄) (2) 기동 훑기 실패 정책 (3) 시험용 Postgres 조달(이 기계에 없다). FD 커밋은 `a14163c`.
**Context**: CONSTRUCTION - queue - NFR Requirements (Step 2~4 완료 · Step 5 답 대기).

---

## NFR Requirements — queue · 답 셋 · 산출물 생성
**Timestamp**: 2026-09-08T23:59:38Z
**User Input**: "Q1 A
Q2 A
Q3 A"
**AI Response**: 답 A · A · A 를 계획 파일에 적었다 — (1) `runs_queued_idx` 부분 인덱스를 `schema.sql` 에 더한다(접점 한 줄) (2) 기동 훑기 실패는 Error 로그 후 기동 계속 (3) 시험용 Postgres 는 사용자가 `docs/testdb-setup.md` 4절대로 직접 깐다. 모호한 답 없음. 산출물 둘을 `construction/queue/nfr-requirements/` 에 냈다 — nfr-requirements.md(규칙 열다섯 판정: 준수 8 · 해당 없음 7 · 조건부 0 · 차단 0 · 오용 사례 · 비-보안 NFR 다섯 축 · 재는 자리 · 진행자에게 넘기는 것 넷) · tech-stack-decisions.md(advisory xact lock 선택과 버린 후보 셋 · 부분 인덱스 · 안 들이는 것 · 차단 게이트 영향). 완료 메시지(2-옵션)를 내고 승인을 기다린다.
**Context**: CONSTRUCTION - queue - NFR Requirements (Step 6·7 완료 · Step 8 승인 대기).

---

## NFR Requirements — queue · 검토 질문
**Timestamp**: 2026-09-09T00:11:04Z
**User Input**: "이 기계에 Postgres가 선다는 의미는, 우선 이 코드를 나중에 병합하는 인원이 다른 서버 컴퓨터에 mediator 서버를 띄울거라서, 그게 가능한거라고 이해돼. 맞아?"
**AI Response**: 반만 맞다고 답했다. 이 기계의 Postgres 는 시험용이다 — `go test` 가 진짜 Postgres 를 요구하고(`internal/store` 는 없으면 t.Fatal) 코드 단계의 시험을 이 자리에서 돌리려는 것. 실제 Mediator 는 띄우는 사람의 서버에 자기 Postgres 를 두고 DATABASE_URL 로 붙는다 — 여기 Postgres 와 무관하다. 여기서 시험을 안 돌리고 CI 만 믿으면 안 깔아도 되나 실패를 코드 단계에서 미리 못 잡는다. 시험은 판을 TRUNCATE 하므로 손으로 띄운 Mediator 와 판을 나누지 않는다(docs/testdb-setup.md 1절).
**Context**: CONSTRUCTION - queue - NFR Requirements (승인 대기 중 검토 문답).

---

## NFR Requirements — queue · 승인 · NFR Design · Infrastructure Design 건너뜀
**Timestamp**: 2026-09-09T00:11:46Z
**User Input**: "승인"
**AI Response**: NFR Requirements 승인을 기록했다. 산출물 둘 · 계획 · 상태 · 감사를 `unit/queue` 에 커밋한다. NFR Design 은 건너뛴다 — 회차 실행 계획이 SKIP 이고 「어떻게」는 Code Generation 계획이 진다(obs 와 같은 근거 · 기술 선택은 tech-stack-decisions.md 가 이미 닫았다). Infrastructure Design 도 건너뛴다 — 새 인프라 0(constraints §6). 다음은 Code Generation Part 1(계획).
**Context**: CONSTRUCTION - queue - NFR Requirements 승인 (Step 9) · NFR Design SKIP · Infrastructure Design SKIP.

---

## Code Generation — queue · Part 1 계획 · 승인 요청
**Timestamp**: 2026-09-09T00:15:09Z
**User Input**: (없음 — NFR 승인의 후속)
**AI Response**: Code Generation Part 1 계획 `construction/plans/queue-code-generation-plan.md` 를 냈다 — 갈래 셋(store · api · mediator) · 못 박는 겉면(FD 에서 둘이 넓어짐: WakeQueued 가 Woken 을 돌려준다 — 승격이 올린 첫 단계 되묻기를 커밋 뒤 알리려고 · nonce 는 store 에 이미 있어 안 옮긴다) · 단계 A1~A10 · B1~B4 · C1~C3 · 시험 열여덟의 이름 · CP0/CP2 재는 순서 · 진행자 표시 넷. 코드 사실 확인: Seal 이 Open 을 스스로 불러 취소된 QUEUED 도 봉인된다 · ClaimStep 은 r.state='RUNNING' 만 집는다 · CreateRun 몸통에 runAcquires·raiseAsks·Records.Open 이 있어 승격도 같은 몸통을 쓴다. NFR 커밋은 `18f3b51`. 이 계획의 승인을 요청한다.
**Context**: CONSTRUCTION - queue - Code Generation Part 1 (Step 4~6 완료 · Step 7 승인 대기).

---

## Code Generation — queue · Part 1 승인 · Part 2 착수
**Timestamp**: 2026-09-09T00:20:49Z
**User Input**: "승인"
**AI Response**: Code Generation 계획 승인을 기록했다. Part 2 를 계획의 순서(A store → B api → C mediator · match 주석)대로 짓는다. DSN 이 아직 없으므로 시험은 짓되 실행은 「미실행」으로 적는다.
**Context**: CONSTRUCTION - queue - Code Generation Part 1 승인 (Step 8·9) · Part 2 시작 (Step 10).

---

## Code Generation — queue · Part 2 생성 완료 · 시험 미실행
**Timestamp**: 2026-09-09T00:33:58Z
**User Input**: (없음 — Part 1 승인의 후속)
**AI Response**: 계획대로 짰다 — A store(신규 `queue.go` · store.go 의 StateQueued·LeaseTTL·createRunIn 추출 · reap.go/claim.go/release.go 의 깨우기 지점 · schema.sql 부분 인덱스 · queue_test.go 아홉) · B api(submit 의 enqueue 둘 · DrainingNodes 합침 · postNodes drain 해제 wake · api_test.go 409→202 셋 · queue_test.go 일곱) · C mediator(wakeQueuedAtStart · LeaseTTL 배선 · 시험 둘) · match.go 주석. 계획 밖 결정 둘을 코드 요약에 적었다 — 큐 기계를 새 파일 queue.go 에 둠(접점 diff 최소화) · 훑기가 FOR UPDATE SKIP LOCKED(취소와의 교착 방지). 검사: go build · go vet · gofmt · glyphscan · 크로스 빌드 넷 · DB 안 쓰는 패키지 시험 통과. cmd/enodectl 의 TestCmdStart_ 셋이 전체 병렬 실행에서 한 번 깨졌으나 단독 재실행과 main 워크트리에서 통과(부하 · 이 유닛 밖). **DB 시험 열여덟과 기존 store/api/mediator 시험 · 커버리지 · CP0 · CP2 는 미실행** — Postgres 가 없다. 코드 요약 `construction/queue/code/code-summary.md`. 완료 메시지(2-옵션)를 내고 승인을 기다린다 — 커밋은 승인 뒤.
**Context**: CONSTRUCTION - queue - Code Generation Part 2 (Step 10~14 완료 · Step 15 승인 대기).

---

## Code Generation — queue · 승인
**Timestamp**: 2026-09-09T00:36:11Z
**User Input**: "승인"
**AI Response**: Code Generation 승인을 기록했다(시험 미실행 상태로의 승인). 코드 · 시험 · 계획 · 코드 요약 · 상태 · 감사를 `unit/queue` 에 커밋한다. 다음은 Build and Test — 진짜 Postgres 가 있어야 CP0 · 커버리지 · CP2 를 돈다. 사용자가 `docs/testdb-setup.md` 4절대로 깔고 DSN 을 주면 돌린다.
**Context**: CONSTRUCTION - queue - Code Generation 승인 (Step 16) · Build and Test 대기(Postgres).

---
