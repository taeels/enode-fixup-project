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

## Build and Test — queue · Postgres 도착 · CP0
**Timestamp**: 2026-09-09T00:47:40Z
**User Input**: "끝"
**AI Response**: Postgres.app(PostgreSQL 18.6) 이 55434 에 서고 DSN 이 붙는 것을 확인했다. 첫 실행에서 둘이 깨졌다 — 시험 픽스처의 단계가 JSON 왕복 뒤 종류를 잃는 것(Agent {} → Run ["true"]) · 409 를 기대하던 네 번째 시험 TestRelease_OnceReleasedAnotherCanTake(놓기 전 202 · 놓으면 기다리던 것이 같은 요청 안에서 RUNNING 으로 고침). 재실행: store · api · mediator 전부 통과. 전체 `go test ./... -coverpkg` — 열여섯 패키지 하한 80% 전부 통과(전체 87.0% · api 80.6% 로 여유 3 문장) · 스킵 0 · glyphscan 통과. 병렬 실행에서 enodectl 셋 · iapadapter 하나가 흔들렸으나 단독 통과 · 이 유닛 밖. 코드 요약 3절을 결과로 바꿨다. 커밋한다. 다음은 CP2 — 실제 노드를 이 기계에 띄워 scene-gates 3절의 명령을 돈다.
**Context**: CONSTRUCTION - queue - Build and Test (CP0 통과 · CP2 준비).

---

## Build and Test — queue · CP2 · 산출물 다섯
**Timestamp**: 2026-09-09T00:52:59Z
**User Input**: (없음 — CP0 의 후속)
**AI Response**: CP2 를 이 기계에서 쟀다 — 판 `enode_cp2` 를 따로 파고 Mediator(127.0.0.1:18080) 와 enode(보드 선언 · 5초 광고)를 띄워 scene-gates 3절 명령을 쳤다. a RUNNING · b 202 QUEUED · status QUEUED · 목록·상세(requires) · a 끝나며 같은 tx 에서 b 승격(로그 순서로 확인) · c 422 → FAILED. 첫 시도의 c 는 계약 입력 형식(속성을 형제 키로 편다)을 틀려 400 이었고 고쳐 다시 쟀다. Build and Test 산출물 다섯을 `construction/build-and-test/` 에 냈다. 커밋한다. 규칙의 물음 「Build and test instructions complete. Ready to proceed to Operations stage?」 — 이 회차에서 Operations 는 자리표시자이고 실제 다음은 PR to main 이다.
**Context**: CONSTRUCTION - queue - Build and Test (CP2 통과 · 승인 대기).

---

## Build and Test — queue · 승인 · PR
**Timestamp**: 2026-09-09T00:59:07Z
**User Input**: "PR 하자. 그 다음 내가 할일은 뭐지?"
**AI Response**: Build and Test 승인으로 받았다. `unit/queue` 를 origin 에 push 하고 `main` 으로 PR 을 연다(roster §5 · 게이트 초록 뒤). 다음 유닛은 drain(W2 · CP3 · 의존 queue) — queue 병합 뒤 착수하되 Functional Design 은 먼저 할 수 있다.
**Context**: CONSTRUCTION - queue - Build and Test 승인 · PR to main.

---

## drain — 착수 · Functional Design 계획
**Timestamp**: 2026-09-09T01:24:15Z
**User Input**: "Queue는 머지 됐어 확인해봐 확인하고 drain 진행하자"
**AI Response**: PR #5 병합을 확인했다(`310c22d` · 2026-09-09T01:00:41Z). `main` 을 fast-forward 하고 `unit/drain` 을 땄다. drain 유닛 정의 · enode-features 3.2.2 · decisions §1 · ADR-063 §2·§3·§4·§6 · scene-gates CP3 · enode 의 advertise/leases/claim(Worker)/paths/config · api postResult · obs 의 contract.Policy 를 읽었다. obs 와 queue 가 이미 낸 것(draining 열 · 매칭 제외 · 해제 깨우기 · Cancel 의 깨우기)을 빼고 남은 셋 — 노드의 정책 파일 읽기와 광고 · 응답 drain 을 Worker 로 · postResult 끝의 at-boundary 취소 — 로 FD 계획 `construction/plans/drain-functional-design-plan.md` 를 냈다. 질문 셋: 정책 파일 위치·이름 · 응답 drain 의 배선(cmd/enode 한 줄 — panel 의 파일) · 파일 권한 검사.
**Context**: CONSTRUCTION - drain - Functional Design (Step 2~4 완료 · Step 5 답 대기).

---

## drain — Functional Design · 답 셋 · 산출물 생성
**Timestamp**: 2026-09-09T01:58:09Z
**User Input**: "Q1 A
Q2 A
Q3 A"
**AI Response**: 답 A · A · A — 정책 파일은 설정 옆 `<stem>.policy.yaml` · 응답 drain 은 Held 로 나르고 `cmd/enode` 한 줄(조율) · 권한은 경고만. 모호한 답 없음. 산출물 셋을 `construction/drain/functional-design/` 에 냈다 — business-logic-model(정책→광고→응답→Worker 흐름 · at-boundary 의 세 tx 와 Worker 가 닫는 창 · 해제 · 파일 행렬) · business-rules(정책 · 경계 · 파일 · 응답 · 보안 확장 판정 · 안 하는 것) · domain-entities(파일 스키마 · Policy/AdvertResponse.Drain/Held/NodeDrain · 어휘 상수를 contract 로 한 벌 · cmd/enode 한 줄). 완료 메시지(2-옵션)를 내고 승인을 기다린다.
**Context**: CONSTRUCTION - drain - Functional Design (Step 6·7 완료 · Step 8 승인 대기).

---

## drain — Functional Design 승인
**Timestamp**: 2026-09-09T02:00:44Z
**User Input**: "승인할게"
**AI Response**: drain FD 승인을 기록했다. 산출물 셋 · 계획 · 상태 · 감사를 `unit/drain` 에 커밋한다. 다음은 NFR Requirements — 새 HTTP 표면 0 · 파일 표면 하나. 물음은 하나(취소 실패의 재시도 자리).
**Context**: CONSTRUCTION - drain - Functional Design 승인 (Step 9).

---

## drain — NFR Requirements · 계획 · 물음 하나
**Timestamp**: 2026-09-09T02:01:17Z
**User Input**: (없음 — FD 승인의 후속)
**AI Response**: NFR Requirements 계획 `construction/plans/drain-nfr-requirements-plan.md` 를 냈다. 실측 표 · 권장값(대기 길이 = 광고 주기 · 오류 로그는 원인이 바뀔 때만 · 캐시 없음) · 물음 하나 — at-boundary 취소가 DB 오류로 실패했을 때 재시도 자리(없음 vs 하트비트).
**Context**: CONSTRUCTION - drain - NFR Requirements (Step 2~4 완료 · Step 5 답 대기).

---

## drain — NFR Requirements · 답 · 산출물 생성
**Timestamp**: 2026-09-09T02:07:02Z
**User Input**: "A"
**AI Response**: 답 A(취소 실패는 로그 · 재시도 없음 · 수락한 위험으로 기록)를 적고 산출물 둘을 `construction/drain/nfr-requirements/` 에 냈다 — 규칙 열다섯 판정(준수 8 · 해당 없음 7 · 조건부 0) · 오용 사례 · 비-보안 NFR 다섯 축 · 진행자 표시 넷 · 기술 선택(yaml.v3 재사용 · 파일 감시 안 들임). 완료 메시지를 내고 승인을 기다린다.
**Context**: CONSTRUCTION - drain - NFR Requirements (Step 6·7 완료 · Step 8 승인 대기).

---

## drain — NFR Requirements 승인 · NFR Design · Infrastructure Design 건너뜀
**Timestamp**: 2026-09-09T02:07:52Z
**User Input**: "승인"
**AI Response**: 승인을 기록하고 커밋한다. NFR Design · Infrastructure Design 은 queue 와 같은 근거로 건너뛴다(어떻게는 Code Generation 계획이 진다 · 새 인프라 0). 다음은 Code Generation Part 1.
**Context**: CONSTRUCTION - drain - NFR Requirements 승인 (Step 9).

---

## drain — Code Generation · Part 1 계획 · 승인 요청
**Timestamp**: 2026-09-09T02:08:56Z
**User Input**: (없음 — NFR 승인의 후속)
**AI Response**: Code Generation 계획 `construction/plans/drain-code-generation-plan.md` 를 냈다 — 갈래 셋(contract·store → enode·cmd/enode → api) · 겉면(어휘 상수 한 벌 · NodeDrain · policyReader · AdvertResponse.Drain · Advertiser.Held · Held 겉면 넷) · 단계 A1~A4 · B1~B6 · C1~C2 · 시험(enode 여덟 · api 넷~다섯 · store 하나) · CP3 재는 순서 · 진행자 표시 셋. 승인을 요청한다.
**Context**: CONSTRUCTION - drain - Code Generation Part 1 (Step 4~6 완료 · Step 7 승인 대기).

---

## drain — Code Generation Part 1 승인 · Part 2 착수
**Timestamp**: 2026-09-09T02:13:33Z
**User Input**: "승인"
**AI Response**: 계획 승인을 기록했다. Part 2 를 A(contract·store) → B(enode·cmd/enode) → C(api) 순으로 짓고 이 기계의 Postgres 로 시험까지 돌린다.
**Context**: CONSTRUCTION - drain - Code Generation Part 1 승인 (Step 8·9) · Part 2 시작.

---

## drain — Code Generation Part 2 완료 · CP0 · CP3
**Timestamp**: 2026-09-09T02:23:47Z
**User Input**: (없음 — Part 1 승인의 후속)
**AI Response**: 계획대로 짰다 — A(contract 상수 · store 별칭 · NodeDrain) · B(policy.go · advertise · leases · claim · cmd/enode 한 줄 · 시험 여덟) · C(postResult 갈래 둘 · drainAtBoundary · 시험 다섯). 첫 실행에서 잡은 것 — 기존 시험이 Held 없이 Worker 를 만들어 SIGSEGV → Drain()·Renew() nil 안전. 되돌림 뒤 취소 갈래를 loopRun 계약으로 덮는 시험을 더해 internal/api 80.1% → 80.5%. CP0 — 열여섯 패키지 80% 통과 · 전체 87.0% · 스킵 0 · glyphscan · 크로스 빌드. CP3 실동작 — at-boundary 통보 · 경계 취소(verdict drain:<node_id> · s1 DONE · record 에 산출) · 202 QUEUED · 해제 후 RUNNING · graceful 완주 · 양쪽 로그. 코드 요약 `construction/drain/code/code-summary.md`. 완료 메시지를 내고 승인을 기다린다.
**Context**: CONSTRUCTION - drain - Code Generation Part 2 (Step 10~14 완료 · Step 15 승인 대기).

---

## drain — Code Generation 승인 · Build and Test · PR
**Timestamp**: 2026-09-09T02:25:48Z
**User Input**: "PR을 열게, 이거 merge하면 이제 태호님이 뭘 해야하지?"
**AI Response**: Code Generation 승인으로 받았다. 코드와 문서를 커밋하고 Build and Test 문서(unit · integration · summary)에 drain 절을 더해 커밋한 뒤 `unit/drain` 을 push 하고 `main` 으로 PR 을 연다. 병합 뒤 nacl1119(문태호)의 일 — panel 유닛(W3 · CP4): 이 FD 가 정한 정책 파일 `<stem>.policy.yaml` 의 `drain` 키를 제어판 토글이 쓴다.
**Context**: CONSTRUCTION - drain - Code Generation 승인 (Step 16) · Build and Test · PR to main.

---

## greet-play — 보드 스피커 재생이 지금 되는지 · 첫 실행
**Timestamp**: 2026-09-09T04:29:09Z
**User Input**: "나 runctl 제출해서, 안녕하세요 손신입니다 tts 소리를 보드에서 붙어있는 스피커로 재생하고 싶음. 가능해 지금 ?" · "wav가 안돼 mp3로 할거야 아마 enode 가 그렇게돼있을거임 b 진행"
**AI Response**: `runctl capabilities` 에 `audio_playback=true` · `board=rpi2b-v1.1` 노드가 있었다(당시 `270c97c94415 taeels@sunnybook`, 속성을 하나씩 눌러 보니 `os=windows` · `host_arch=amd64`). 역할 둘(`tts` 가 mp3 를 만들고 `board` 가 튼다)을 속성으로 부르는 계약을 짜서 dry-run 으로 두 노드가 걸리는 것을 확인하고 제출했다 — `greet-play-20260909-132909` SUCCEEDED, `player=MediaPlayer` · `played seconds=1.959`(클립 1.933). 어느 스피커로 나갔는지는 라벨로 안 갈린다고 적어 뒀다. 문장은 고정 문장이 아니라 명시 문장이라 `say` 를 직접 불렀다(tts_workspace/CLAUDE.md 규약).
**Context**: 별개 — greet-play recipe (AI-DLC 유닛이 아님).

---

## greet-play — recipe 를 저장소에 둔다 · PR #13
**Timestamp**: 2026-09-09T04:38:32Z
**User Input**: "이 recipe runctl 계약서를 알려줄 수 있어? 그거 enode-fixup-workshop에 디렉토리 하나 파서 커밋올려주면 될듯" · "응 올려, 그리고 PR 최신에서 그 커밋만 포함되게 pr 올릴수있나, greet-play만 변경사항으로 되게" · "enode-fixup-project 말한거였어. 이 컴퓨터에서 이 디렉토리 찾아봐" · "enode-fixup-workshop은 안건드려도되고"
**AI Response**: 처음에 `enode-fixup-workshop` 에 올렸다(PR #3 · 브랜치 · `shin-son-development` 의 커밋 하나). 사용자가 그쪽은 그대로 두라고 해서 손대지 않았다. `enode-fixup-project` 에 `recipes/greet-play/`(README · greet-play.json · submit.sh)를 두고 `main` 에서 딴 `recipe/greet-play` 로 PR #13 을 열었다(변경 3 파일 · 커밋 1). 이후 병합됐다.
**Context**: 별개 — greet-play recipe.

---

## greet-play — 소리가 난 자리가 보드가 아니었다
**Timestamp**: 2026-09-09T05:07:28Z
**User Input**: "근데 그 문장이 윈도우 컴퓨터에서 나온거야 설마?" · "다시 한번 런해봐 내가 오디오 를 꺼놈;;" · "다시 한번 제출해봐 , 그리고 왜이렇게 빨리 말하지?"
**AI Response**: 맞다고 답했다 — `play` 스텝은 `os=windows` 노드 위에서 .NET `MediaPlayer` 로 그 기계의 기본 오디오 장치에 냈고, 파이는 별개 컴퓨터라 보드 스피커에 닿을 경로가 없었다. 그때까지 세 번(04:29 · 05:01 · 05:02Z) 남의 기계에서 소리가 났다는 것과, 어디서 나는지 모르는 채로 세 번 돌린 것이 잘못이라고 적고 추가 실행을 멈췄다. 사용자가 오디오를 켜고 다시 요청해 두 번 더 돌렸다(05:06 · 05:07Z, 같은 결과). 빠른 이유는 `say` 기본 속도이고 Yuna 가 다른 여덟 음성보다 40% 빠르다는 것을 실측으로 보였다.
**Context**: 별개 — greet-play recipe.

---

## tts — Typecast 로 바꾸고 상현으로 고정 · 라벨 갱신 · CLAUDE.md
**Timestamp**: 2026-09-09T05:38:15Z
**User Input**: "그냥 typecast 다운받아서 그거로 해주라." · "무슨 키?" · "key 어디에다 적으면 되니 ?" · "음성은 상현으로 픽스하고, 캐패블리티 업데이트 하자" · "고쳐."
**AI Response**: Typecast 는 내려받는 엔진이 아니라 API 라 계정 · 키 · 글자당 과금이 따라온다고 알렸고, tts_workspace/CLAUDE.md 의 외부 TTS 금지 조항과 어긋난다는 것을 두 번 물었다. 사용자가 키를 `~/.config/typecast/api.key` 에 두었다. `bin/tts-typecast.sh` 를 만들었다 — `X-API-KEY` · `ssfm-v30` · `kor` · 응답이 오디오 바이트라 HTTP 코드로 걸러냄 · 걸러진 환경에서 `HOME` 을 못 믿어 `id -un` 으로 집을 되짚음. 590 음성 중 다섯을 사용자 맥에서 순서대로 들려주고 상현(`tc_69fc0cff784968297fb45daa`)을 기본값으로 박았다. 라벨 `tts_typecast: "yes"` · `voice_typecast: Sanghyun` · `typecast_model: ssfm-v30` 을 더했다(`tts: say-macos` 는 그대로 — 바꾸면 기존 계약이 못 닿는다). enode 재기동은 분류기에 막혀 사용자가 직접 했고 `capabilities` 에 실린 것과 dry-run 매칭을 확인했다. "고쳐." 로 CLAUDE.md 를 두 엔진 서술로 고쳤다(기본은 여전히 `say`, Typecast 는 명시로 부를 때만).
**Context**: 별개 — tts 노드 (tts_workspace, 저장소 밖).

---

## greet-play — 계약을 Typecast 로 · 다시 재생
**Timestamp**: 2026-09-09T05:47:13Z
**User Input**: "그거 바꿔줘. 그리고 바꾼뒤에 runctl 제출해서 보드에서 소리 나는지 다시한번 확인해볼래"
**AI Response**: `make` 스텝을 `bin/tts-typecast.sh "$1" "$OUT/greeting.mp3"` 로, `requires` 를 `tts_typecast=yes` + `voice_typecast=Sanghyun` 으로 바꿨다(`fmt_mp3` 는 이제 불필요). `greet-play-20260909-144713-70963` SUCCEEDED — 44.1kHz · 85,681 바이트 · 2.14초로 `say` 경로에서는 나올 수 없는 값이라 바이트가 Typecast 를 거쳤다. 소리는 여전히 sunnybook 의 기본 장치였다.
**Context**: 별개 — greet-play recipe.

---

## greet-play — pi-stage 갈래를 보고 판단을 정정한다
**Timestamp**: 2026-09-09T06:19:09Z
**User Input**: "https://github.com/taeels/enode-fixup-project/compare/main...recipe/pi-stage 이 브랜치보고 너가 준 두번째 답장을 다시 검토해. 1번째는 안돼" · "응 --pi로 상현 목소리 돌려봐"
**AI Response**: `ssh` 로 파이에 넘기는 길을 "기록에 안 남고 exit 0 을 못 믿는다" 며 덜 좋은 길이라 했던 판단을 철회했다 — 브랜치의 `--pi` 갈래는 커널 재생 카운터(`/proc/asound/card1/pcm0p/sub0/status`)를 찍어 실제 재생을 증명하고 그것이 Record 에 남으며, 매칭(노드)과 라우팅(스피커)이 다른 층이라는 것도 그쪽이 맞았다. `greet-play-pi.json` 의 `make` 만 Typecast 로 갈아 끼워 제출했다 — `play` 가 `scp failed rc=255`(1초 만에). 그때 파이가 자리를 옮기는 중이었다.
**Context**: 별개 — greet-play recipe.

---

## greet-play — 보드가 nacl1119 노드로 옮겨감 · rc=255 를 좁혀 파이 잭까지
**Timestamp**: 2026-09-09T06:42:09Z
**User Input**: "지금 보드를 taeho nacl1119@DESKTOP-5DRAFHN 여기에 보드 연결 바꿨을거임, 태호가 지금 enode에 cap 등록중" · "광고에 실렸대, 다시 한번 보자." · "똑같대 다시해봐" · "태호님은 키옵션 없이 그냥 붙는거같네, 나도 다시해보자" · "키 심었대 다시 돌려봐" (태호의 `ssh -v` 로그와 키 생성 로그를 같이 붙임)
**AI Response**: 라벨이 `9b3586dc2564 nacl1119@DESKTOP-5DRAFHN` 로 옮겨간 것을 dry-run 으로 확인했고 계약은 노드 이름을 안 적어 손댈 것이 없었다. `--pi` 가 네 번 `rc=255`. 처음엔 키 부재로 봤다(`-v` 로그의 `identity file ... type -1` 일곱 개) — 키를 심고 `BatchMode=yes` 로 `whoami` 가 붙은 뒤에도 같았다. 소리 없는 진단 계약을 여섯 번 던져 좁혔다: 스텝은 MSI 계정 · 세션 7 · `HOME` 없음 · 콘솔 없음; **Windows 내장 `ssh.exe` 는 `ssh -V` 조차 rc=255 에 출력 0 바이트**(stdin `NUL` · `Start-Process` 새 콘솔 · 전체 경로 모두 무효); 같은 기계의 `C:\Program Files\Git\usr\bin\ssh.exe` 는 정상이고 그것으로 `whoami` · scp 왕복 · 파이의 카드 배치(`0 vc4hdmi` · `1 bcm2835 Headphones`)까지 됐다. `~/.ssh/config` 가 그 주소를 `id_rpi_sunnypi` 에 고정하고 있어 새 키는 쓰이지도 않았다. `play` 스텝이 Git 의 ssh · scp 가 있으면 그것을 쓰도록 고쳐 제출 — `greet-play-pi-20260909-154201-76258` SUCCEEDED, `mpg123 rc=0`, 스텝 4.5초에 클립 2.43초. 사용자가 그 자리에서 상현 목소리를 들었다. 원인은 못 밝혔고 증상과 우회만 적었다.
**Context**: 별개 — greet-play recipe.

---

## greet-play — 세 사람 인사 · PR #21 · 이 기록
**Timestamp**: 2026-09-09T06:55:16Z
**User Input**: "아까 소리 확인했어. 이 내용 PR 해야할거같고. 태양님, 태호님, 태완님 안녕하세요 라고 목소리 출력도 다시해줘 runctl로" · "지금 소리도 나니까, 관련한 계약 내용들하고 지금 성공했던 부분들, 디버깅 내용들 업데이트하면 될 거 같아. enode-fixup-project 폴더야."
**AI Response**: `greet-play-pi-20260909-154417-76559` SUCCEEDED — "태양님, 태호님, 태완님 안녕하세요" 3.03초, 파이 잭. PR #21 `recipe/pi-stage-typecast` → `recipe/pi-stage`(base 를 main 이 아니라 pi-stage 로 잡은 이유 — `--pi` 갈래가 아직 main 에 없어서 main 으로 내면 taeels 의 커밋이 딸려 간다): `make` Typecast(host · pi 둘 다) · `play` Git ssh 우선 · README 에 진단 경위. 이어 `tts-node/`(tts-typecast.sh 사본 · labels.yaml · 세우는 법)와 `diag-node.json` + `submit.sh --diag`(진단 여섯 번을 하나로 접음)를 얹었다. 이 감사 로그와 `aidlc-state.md` 는 `main` 에서 딴 `shin-son/greet-play-record` 로 따로 PR — 담당 기록이 taeels 의 브랜치를 타지 않게.
**Context**: 별개 — greet-play recipe. 열린 것 — 내장 ssh.exe 가 enode 스텝 안에서 뜨지 않는 원인.

---
