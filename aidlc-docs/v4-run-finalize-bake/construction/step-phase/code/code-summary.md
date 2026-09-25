# `step-phase` — 코드 요약

계획은 `construction/plans/step-phase-code-generation-plan.md` (단계 열여섯 · 승인
2026-09-25T09:32:03Z), 설계는 `construction/step-phase/functional-design/` 의 셋이다.
아래의 **FD** 는 Functional Design 산출물의 줄임이다 — 「FD 규칙 1절」은 `business-rules.md` 1절이다.

- **브랜치** `unit/step-phase` · 기준 `242ca33`
- **패키지 넷** `internal/contract` · `internal/store` · `internal/api` · `internal/record`. 노드 쪽 diff 0 ·
  `internal/match` diff 0 · 새 import 0 (표준 라이브러리 `net/url` 을 시험 파일 하나가 쓴다)
- **DB 스키마** `steps` 에 칸 셋 (`phase` · `phase_since` · `exit`). 기동 때 `ALTER TABLE … ADD COLUMN IF NOT EXISTS`

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/contract/result.go` | 새 (211 줄) | 결과 어휘 — `Exited` · `Outcome` · `Stage` · `Diagnostics` · `CollectNote` · `CheckpointCapture` · `BuildManifest` · `BuildRecord` · `Pinned` · `MergeResult` · `MergeOps` 와 상수 · `Exited.Check` |
| `internal/store/schema.sql` | 고침 (+18) | 칸 셋과 주석 한 덩어리 |
| `internal/store/claim.go` | 고침 (+241 −5) | phase 상수 셋 · claim 과 재전달이 phase 를 적는다 · `MarkExited` · `classifyExit` · `sameExit` · `ErrNoSuchStep` · `ErrExitRejected` · `exitRejection` · `StepResult` 새 칸 아홉 · `OutOfVocabulary` · `Claimed` 칸 일곱 · `fillFromContract` |
| `internal/store/rollback.go` | 고침 (+4 −2) | 되돌림 둘(loop · validate_with)이 칸 셋을 지운다 |
| `internal/store/ask.go` | 고침 (+2 −1) | 계획 거절의 되감기가 칸 셋을 지운다 |
| `internal/store/observe.go` | 고침 (+29 −2) | `StepView` 칸 셋 · `Steps` 의 SELECT · `RequireView.Candidates` |
| `internal/store/queue.go` | 고침 (+69) | `Candidates` · `CandidatesFor` |
| `internal/store/seal.go` | 고침 (+21 −4) | `StepFiles` 가 Record 의 새 칸 넷을 채운다 |
| `internal/record/record.go` | 고침 (+13) | `StepFile` 칸 넷 |
| `internal/api/api.go` | 고침 (+71) | 라우트 한 줄 · `postExited` · `postResult` 의 경고 로그 · `runView.CandidatesAt` · `getRun` 의 후보 수 |
| 시험 넷 | 새 | `internal/contract/result_test.go` · `internal/store/exited_test.go` · `internal/store/candidates_test.go` · `internal/api/exited_test.go` |

파일 행렬(`unit-of-work-file-matrix.md` 1.1절) 밖의 소스는 계획 2절이 적은 넷 그대로다 — `result.go` ·
`rollback.go` · `ask.go` · `seal.go`. 그 밖은 0 이다. 시험 파일은 모두 새 파일이고 기존 시험 파일은 안 고쳤다.

---

## 2. 규칙이 어디에 있나

```text
   규칙 (FD business-rules.md)                  자리
   1절  종료 보고의 수락 표 · 409 문구             store.MarkExited (한 UPDATE) · store.classifyExit (0 행일 때)
        같은 키의 재전송에서 본문이 다르면 경고     store.sameExit
        응답 코드로 옮기기                        api.postExited
   2절  본문 검사 (400)                          contract.Exited.Check.  파싱 오류와 seq 는 api.postExited
   3절  칸마다의 시계                             store.StepView 의 godoc · record.StepFile 의 주석 · schema.sql 주석
   4절  phase 의 전이                            claim.go ClaimStep · redeliver 의 UPDATE ·
                                                rollback.go 둘 · ask.go rewindToPlanner
   5절  result 새 칸은 값으로 거절 안 함           store.StepResult.OutOfVocabulary · api.postResult 의 경고
   6절  Record 의 단계 기록                       store.StepFiles
   7절  진행 조회에 싣는 조건                      store.Steps (CLAIMED 가 아니면 phase 를 비운다) · api.getRun
   8절  후보 수                                  store.CandidatesFor
   9절  Claimed 의 계약 칸                        store.fillFromContract
```

**수락은 한 문장이다.** 받을 조건 전부를 UPDATE 의 WHERE 에 둔다. 0 행이면 `classifyExit` 가 Run 과 단계를
LEFT JOIN 한 번으로 읽어 표의 1 ~ 12 중 처음 맞는 줄을 고른다. 행을 바꾸지 않는다.

---

## 3. 계획 4절의 결정 다섯 — 지은 모양

```text
   ①  409 문구         exitRejection 이 문구를 나르고 Is(ErrExitRejected) 가 참이다.
                       api 는 err.Error() 를 그대로 reason 에 싣는다
   ②  어휘 밖 값        StepResult.OutOfVocabulary 가 (칸, 값) 쌍을 돌려준다.  빈 값은 제외.
                       api.postResult 가 쌍마다 "result carries an unknown value" 경고 한 줄
   ③  재전송 경고       "exit report repeated with a different outcome or time; the first one stands".
                       시각은 DB 정밀도(마이크로초)로 맞춰 비교한다
   ④  DB 오류           postExited 는 503 "query failed" — 노드가 재시도를 멈추지 않는다
   ⑤  시각의 모양       Record 의 exited_at · finalized_at 은 started_at 과 같은 UTC RFC 3339 나노초 문자열
```

---

## 4. 계획과 다르게 된 자리

- **조각 0 의 라우트 수** — 계획 2절의 표는 `internal/api/exited_test.go` 에 「조각 0 의 라우트 수」를
  적었다. FD 흐름 7절은 「라우트 수를 세는 시험은 없다」다. 시험은 라우트가 POST 로 등록됐는지(다른
  메서드는 405)만 확인하고, 수는 Step 14 의 `grep -c 'mux.HandleFunc' internal/api/api.go` = **19** 로 확인했다
- **벤치마크의 도우미** — `obsStore` 의 도우미(`scratchDB` · `replaceDBName`)가 `*testing.T` 만 받아서,
  `candidates_test.go` 에 벤치마크 전용 `benchStore` 를 두었다. 다른 시험 파일을 고치지 않으려고 30 줄을 새로 썼다

---

## 5. 코드 검사 (`unit-of-work.md` 0절 · CI 와 같은 명령)

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go build ./...` | exit 0 |
| `go test ./... -count=1` (`-coverpkg=./...` · `-json`) | 통과 1,960 · 실패 0 · 스킵 0 (U1 병합 때 1,891) |
| 패키지마다 커버리지 (CI 의 awk 그대로) | 스무 패키지 전부 80% 이상 · 전체 85.7%. `internal/store` 83.1% · `internal/api` 82.5% · `internal/record` 84.4% · `internal/contract` 92.5% |
| 새 함수 | `Exited.Check` · `OutOfVocabulary` 100% · `StepFiles` 96.9% · `classifyExit` 95.5% · `fillFromContract` 88.6% · `postExited` 87.5% · `CandidatesFor` 80.8% · `MarkExited` 77.8% · `getRun` 78.3% |
| 크로스 빌드 셋 | windows/amd64 · linux/arm (GOARM=7) · darwin/arm64 exit 0 |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| U+2605 · `glyphscan` | 0 · 「135 files scanned, no decorative glyph」 |
| `golangci-lint` v2.13.2 | 39 건 — U1 병합 뒤 `main` 과 같다. 이 유닛이 고친 파일의 경고 0 |
| **조각 0** (기동이 안 깨졌다) | build · vet · test 통과 · `mux.HandleFunc` 18 -> 19. ADR-073 (제품 게이트)의 기존 시험 회귀 0 |
| `cmd/enodectl/probe.lock` | 측정이 바꿔 되돌렸다 (Reverse Engineering 이 적은 알려진 부채) |

**시험이 안 닿는 줄** — 저장소 오류의 갈래(DB 가 문장을 거절할 때)와 `getRun` 에서 후보 수를 못 셀 때의 갈래다.
뒤의 것은 warnings 에 한 줄을 싣는 자리인데, HTTP 시험에서 그 한 트랜잭션만 실패시키는 방법이 없다. 모양은
계약을 못 읽을 때의 갈래(`TestGetRun_SaysSoWhenTheContractCannotBeRead`)와 같다.

**시험이 결함을 잡는지 한 번 확인했다.** `ask.go` 의 되감기가 칸을 안 지우게 바꾸면
`TestPhase_RewindsClearTheColumns` 가 `run-rewind#1 kept {Phase:finalizing ...}` 로 깨진다. 되돌리면 지난다.

---

## 6. 대기 조회의 비용 — NFR 을 건너뛰어 계획 3절이 받은 것

`BenchmarkCandidatesFor` — 광고 50 · 요구 줄 3 · 임대 10 · drain 5. 이 기계(Intel N100 · Docker 안의
Postgres 17 · 같은 호스트)에서 `-benchtime=2s -count=3`.

```text
   692,600 ns/op    806,021 ns/op    750,295 ns/op      조회 한 번에 약 0.75 ms
```

세 표를 한 트랜잭션에서 한 번씩 읽는 비용이 대부분이다. QUEUED 인 Run 하나를 `GET /v1/runs/{id}` 로
조회할 때만 붙고, 목록과 제출 응답에는 안 붙는다. 상한은 두지 않았다 (계획 3절 ②).

**인스턴스 대조** (계획 3절 ①) — 수락 표 10 · 11 을 저장소 시험(`TestMarkExited_TheAcceptanceTable`)과
HTTP 시험(`TestExited_TheRouteAndItsAnswers` 의 「a restarted node」 · 「another node」) 둘 다에서 409 로 확인했다.

---

## 7. 병합 뒤 — 뒤 유닛이 들어오기 전

```text
   옛 노드 (오늘의 enode)    exited 를 안 보낸다.  phase 는 claim 때의 running 에 머물고 Record 에
                            last_phase: running 만 남는다.  result 의 새 칸도 없다
   진행 조회                 CLAIMED 단계마다 phase 와 phase_since 가 새로 보인다.  exit 는 아직 안 나온다
   QUEUED 조회               requires[].candidates 와 candidates_at 이 새로 보인다 — 오늘 바로 쓸 수 있다
   claim 응답                계약이 새 칸을 적었으면 실린다.  오늘의 노드는 모르는 칸이라 버린다
```

진행 조회의 시각(`phase_since` · `started_at`)은 DB 연결의 시간대로 표기된다 — 오늘의 `started_at` 과 같다.
시험은 글자가 아니라 시각으로 비교한다.

---

## 8. 뒤 유닛에 넘기는 것 (FD 흐름 10절 그대로)

```text
   finalize     exited 를 보낸다 — 종료 status 가 정해진 뒤 · 결과 확정 전 · 한 번.
                200 이면 멈춘다 (accepted 가 false 여도).  409 이면 멈춘다.  404 · 405 면 그 Run 동안
                다시 안 보낸다.  5xx · 끊김은 결과 확정과 나란히 재시도한다.
                result 에 exited_at · finalized_at · finalize · upload · reason · diagnostics 를 채운다.
                diagnostics 는 contract.Diagnostics 로 옮겨 담는다 (enode.HarvestNote -> CollectNote)
   checkpoint   result 에 checkpoint_capture 를 채운다
   bake         build 단계의 exited — 마지막 명령이 끝난 뒤 한 번.  merge 단계는 안 보낸다.
                result 에 build (contract.BuildManifest) · merge (contract.MergeResult) 를 채운다
   모두         contract/result.go 의 모양은 더하기만 한다
```

---

## 9. 정본에 되돌려 올릴 것 (FD 흐름 11절 · 진행자가 올린다)

```text
   mediator-api.md exited 절     경로 /v1/runs/{run}/steps/{seq}/exited · 본문의 node · instance ·
                                응답 {run_id, seq, accepted} · 409 의 다섯 경우와 문구 · 400 의 조건 ·
                                merge 단계는 200 으로 받지 않는다 · DB 오류는 503 (이 계획이 더했다)
   mediator-api.md 진행 조회 절   phase · phase_since 는 CLAIMED 일 때만 · phase_since 의 시계가 phase 에
                                따라 바뀐다 · exit 는 상태와 무관 · QUEUED 의 candidates 셋과 candidates_at
   mediator-api.md result 절     새 칸 아홉과 그 타입 · 값으로 거절 안 함
   ADR-075 §4 의 봉투            receipt 묶음이 wire 에서는 평평한 칸이 됐다
   Record 형식 문서              단계 기록의 exited_at · finalized_at · exit · last_phase 와 칸마다의 시계
```

---

## 10. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 첫 줄(종료 보고의 인스턴스 대조)은 수락 표 10 · 11 이 닫고 6절의 시험이 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |
