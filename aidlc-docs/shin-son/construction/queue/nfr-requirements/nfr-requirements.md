# queue — 지켜야 하는 것

AI-DLC Construction · queue 유닛(W1 · CP2)의 NFR Requirements 산출물 하나다.
**무엇을 지켜야 하는가**이고 어떻게는 여기가 아니다. 기술 선택은
`tech-stack-decisions.md`, 계획과 답 셋은
`aidlc-docs/shin-son/construction/plans/queue-nfr-requirements-plan.md`.

정본 — 확장 `security-baseline.md`(차단성) · 팩 `requirements/decisions.md` 1절 · 2절 ·
3절 · `constraints.md` · 이 유닛의 Functional Design 산출물 셋 · obs 의 NFR 표(잇는다).
**어긋나면 `enode-design` 이 이긴다** (`requirements/canon.md` 머리말).

---

## 1. 답 셋이 부른 것

물음 셋의 답이 **A · A · A** 다.

```text
   답      정한 것                                        이 유닛이 지는 일
   ─────   ────────────────────────────────────────────   ───────────────────────────────
   1 = A   runs_queued_idx ON runs (created_at)            schema.sql 한 줄.  접점(진행자 직렬)
           WHERE state = 'QUEUED'  부분 인덱스              이 유닛의 스키마 변경이 0 에서 1 이 된다
   2 = A   기동 훑기 실패는 Error 로그 · 기동 계속           main.go 의 갈래 하나.  reaper 와 같은 결
   3 = A   시험용 Postgres 는 사용자가 직접 깐다             docs/testdb-setup.md 4절.  DSN 을 받으면
                                                           Code Generation 의 시험과 CP0 를 여기서 돈다
```

**Functional Design 의 파일 목록이 하나 는다** (`domain-entities.md` 6절 재확인표).

```text
   internal/store/schema.sql   CREATE INDEX 한 줄 — 답 1=A     접점 (obs 가 이미 둘을 더했다 · 진행자 직렬)
```

---

## 2. security-baseline — 열다섯 규칙의 판정

obs 의 표를 잇는다. **이 유닛은 새 HTTP 표면을 안 만든다** — 기존 `POST /v1/runs` 의
응답 코드 하나(202)가 늘고 store 함수가 는다. 그래서 obs 가 데모 무인증 읽기 셋 때문에
조건부로 둔 08 · 12 가 여기서는 해당 없음이다 — 같은 규칙에 다른 판정이고 근거를 적는다.

| 규칙 | 판정 | 근거 |
|---|---|---|
| SECURITY-01 암호화 | 해당 없음 (기존 코드 사실) | 저장소를 새로 안 만든다. `decisions.md` 3절이 기록만으로 닫았다 |
| SECURITY-02 중간자 접근 로깅 | 해당 없음 | 로드밸런서 · 게이트웨이 · CDN 이 없다 |
| SECURITY-03 애플리케이션 로깅 | 준수 (새 로그 범위) | 새 로그 셋 — 대기 진입 Info · 승격 Info · 기동 훑기 실패 Error. 전부 `run_id` · 노드 · 수만 싣고 계약 본문 · principal · submitter 를 안 찍는다. 로거는 기존 `s.log` · `Store.Log`. 중앙 로그 · 상관관계 ID 가 없는 것은 기존 코드 사실(3절 기록만) |
| SECURITY-04 HTTP 보안 헤더 | 해당 없음 | JSON 라우트뿐이고 HTML 을 안 낸다. 새 라우트도 없다 |
| SECURITY-05 입력 검증 | 준수 | 새 입력이 0 이다. `QUEUED` 는 서버가 정하는 값이고 제출 본문 검증은 기존 `Validate` · `io.LimitReader` 그대로. SQL 은 전부 파라미터 바인딩 — 승격 경로의 문장 셋(UPDATE runs · INSERT leases · INSERT steps)도 `CreateRun` 의 것을 뽑아 쓴다. 잠금 키는 상수라 사용자 입력이 아니다 |
| SECURITY-06 최소 권한 | 해당 없음 | IAM · 클라우드 정책 대상이 아니다 (`constraints.md` 6절) |
| SECURITY-07 네트워크 구성 | 해당 없음 (기존 코드 사실) | 새 리스너 없음 |
| SECURITY-08 접근 제어 | 준수 | `POST /v1/runs` 는 `s.auth` 그대로 — 데모 모드에서도 잠긴 채다(obs 의 `read` 래퍼는 읽기 셋에만 걸린다). 202 는 인증 경계를 안 바꾼다. **obs 의 조건부는 읽기 셋의 것이고 이 유닛의 표면이 아니다** |
| SECURITY-09 하드닝 | 준수 | 새 자격증명 0. 오류 응답은 기존 `fail(w, code, reason)` 고정 문구. 승격 실패는 응답이 아니라 로그다 |
| SECURITY-10 공급망 | 준수 | **새 의존 0.** advisory lock 은 PostgreSQL 의 SQL 함수라 `go.mod` 가 안 움직인다 (`tech-stack-decisions.md` 2절) |
| SECURITY-11 보안 설계 | 준수 | 큐 변경이 store 의 함수 넷에 격리되고 api 는 부르기만 한다(관심 분리). 방어 층 — 인증(auth) + 계약 검증(Validate) + tx·잠금(I5). 오용 사례 하나를 3절에 적는다. **요청 한도는 이 유닛의 표면이 아니다** — `POST /v1/runs` 는 인증 뒤이고 데모의 제출은 demo-back 이 자기 라우트로 낸다 |
| SECURITY-12 인증 · 자격증명 | 해당 없음 | 새 자격증명 · 로그인 흐름 0. obs 의 조건부(무인증 읽기 셋)는 이 유닛에 없다 |
| SECURITY-13 무결성 검증 | 준수 | 역직렬화는 기존 `encoding/json` 고정 타입. 승격은 상태 변경이지만 감사 자리는 Record 다 — `QUEUED -> RUNNING` 의 시각은 `leases.granted_at` 과 승격 로그에 남고, 종료 시 봉인이 `assigned` 를 진다. 취소된 QUEUED 도 봉인된다(FD rules §3) |
| SECURITY-14 경보 · 모니터링 | 해당 없음 (범위 밖) | 인증 실패 이벤트가 이 유닛에 없다. 로그 보존 · 대시보드는 `constraints.md` 6절이 뺀 운영. **기록이지 통과가 아니다** (obs 와 같다) |
| SECURITY-15 예외 처리 | 준수 | 모든 DB 호출에 오류 분기. 실패는 닫는 쪽 — 승격 오류는 tx 롤백(부분 승격 0 · 정산도 함께 되돌아온다 · FD logic §5), `CreateQueuedRun` 오류는 503(오늘의 `create failed` 갈래), 잠금은 xact 범위라 롤백에 풀린다(자원 정리). 기동 훑기 실패는 **답 2=A 로 기동을 계속**한다 — 「fail closed」의 반대로 보이지만 열리는 것이 없다: QUEUED 는 아무것도 안 쥐고 다음 지점이 다시 본다. 사용자 문구는 일반 문구 |

**조건부가 0 이다.** 차단 판정도 0 이다.

---

## 3. 오용 사례 하나 (SECURITY-11 의 요구)

```text
   토큰을 가진 클라이언트가 같은 능력을 요구하는 계약을 run_id 만 바꿔 루프로 낸다.
   후보는 있고 전부 busy 라 매번 202 · QUEUED 다.  runs 에 대기 행이 쌓인다.

   막는 것
     ①  auth — 토큰 없이는 못 낸다.  토큰은 함대의 신뢰 경계다 (ADR-010)
     ②  같은 run_id 는 200 + 기존 Run (멱등).  run_id 를 바꿔야 는다
     ③  훑기 비용은 행당 match 한 번이고 잠금 아래 직렬이다 — 폭주해도 정산 요청이
        기다릴 뿐 잘못 승격되지 않는다.  부분 인덱스(답 1)로 빈 큐의 비용은 0 이다

   남는 것
     대기 상한이 없으므로(정본) 행은 계속 는다.  출구는 cancel 뿐이다.  이것은
     수락한 것이다 — 상한은 정책이고 정책 틀을 안 만든다 (constraints §2).
     토큰을 가진 쪽은 이미 신뢰 경계 안이다.  진행자에게 남긴다 (6절)
```

---

## 4. 비-보안 NFR

### 4.1 성능

수치 목표를 안 적는다(계획 2.2). **비용이 어디서 나는지**와 무엇이 누르는지를 적는다.

```text
   빈 큐의 훑기       임대 해제마다 한 번.  답 1=A 의 부분 인덱스가 비어 있어 인덱스 조회 하나.
                     정산 · 취소 tx 에 더해지는 것이 그 한 줄이다

   찬 큐의 훑기       행 수 x match.Match (광고 수 x 요구 수 · 순수 함수 · 메모리).
                     행마다 승격 시 문장 셋.  대기 행은 사람 손이 낸 수다 —
                     데모에서 열을 안 넘고, 실 함대에서도 그 규모다

   잠금 대기          정산 · 취소 · 부분 반납 · 대기 진입이 한 키 뒤에 선다.
                     임계 구간은 훑기 하나(ms).  claim 롱폴은 이 잠금을 안 잡는다 —
                     임대를 지우지 않으므로

   submit 의 추가 비용   DrainingNodes 질의 하나 (nodes WHERE draining <> '').
                     노드 수가 한 자리다.  dry-run 은 안 낸다

   재는 자리          CP2 의 「a 가 끝난 뒤 runctl status <b> -> RUNNING」이 지연 0 을 잰다
                     (동기 · 같은 tx).  비용은 go test -bench 를 안 만든다 — 팩에 없는 값이다
```

### 4.2 확장

```text
   수평 확장 안 한다     Mediator 하나 (constraints §6).  advisory lock 은 DB 안의 것이라
                       인스턴스가 늘어도 직렬화는 유지된다 — 늘릴 계획이 없다는 것과 별개로
                       그 값이 맞다
   커넥션 풀            기본값.  훑기는 부르는 tx 의 커넥션.  WakeQueuedNow 만 하나 더
   대기 행 수           상한 없음 (정본).  부분 인덱스 크기 = 대기 행 수
   runs 표             인덱스가 셋이 된다 (work · created · queued 부분).  INSERT 마다 셋을
                       갱신하되 부분 인덱스는 QUEUED 행에만 든다 — 승격 UPDATE 가 그 행을 뺀다
```

### 4.3 가용

```text
   목표 수치 없다        단일 Mediator (계획 2.2)
   기동 훑기 실패        답 2=A — Error 로그 · 기동 계속.  다음 지점 · Reap 에서 잡힌다
   훑기 tx 오류          호출자 tx 롤백.  정산이 되돌아오면 enode 의 결과 보고가 409/503 을
                       받고 다시 온다 — 오늘의 재보고 경로 그대로.  임대는 안 지워졌으므로
                       회수(Reap)가 뒤를 받친다
   Mediator 크래시       QUEUED 는 임대 0 · 노드 0 이라 되돌릴 것이 없다.  기동 훑기가 죽어 있는
                       동안 풀린 것을 본다 (ADR-064 §6)
   DB 잠금 교착          xact lock 하나뿐이고 잡는 순서가 한 곳(훑기 첫 줄)이라 순환이 없다.
                       claim 의 SKIP LOCKED 와는 다른 행 · 다른 키다
```

### 4.4 신뢰

```text
   불변식               I5 — 승격의 임대 INSERT 가 하나라도 실패하면 전체 롤백.  잠금 아래서는
                       unique 위반이 안 나고, 나면 버그다 — 테스트가 그것을 잡는다
   멱등                 같은 run_id 재제출 200.  WakeQueued 를 두 번 불러도 둘째는 0 행 승격
   FIFO                 created_at, run_id 순.  같은 tx 안에서 앞 행이 잡은 노드는 뒤 행의 busy
   정산 보호            SettleIfDone 은 RUNNING 만 (FD rules §2).  테스트로 잡는다
   시험                 internal/store 에 queue_test.go — 두 자리 · 승격 · FIFO 건너뛰기 · 취소 ·
                       재기동 훑기 · 잠금 경쟁(고루틴 둘).  internal/api 에 409->202 셋 수정 +
                       CP2 시나리오.  cmd/mediator 에 기동 훑기(성공 · 실패 로그).  전부 진짜
                       Postgres — 답 3=A 로 이 기계에 선다
```

### 4.5 유지보수

```text
   커버리지             internal/store 가 하한 80% 에 걸린다.  새 문장은 전부 DB 시험으로
                       덮인다 — 덮이는 문장은 비율을 올린다.  internal/api 의 수정 셋은 기존
                       시험을 고치는 것이라 문장 수가 거의 안 는다.  재는 자리 ci.yml:269
   스킵 0               DB 없이는 t.Fatal 이지 t.Skip 이 아니다 (기존 규약 그대로)
   한 벌                CreateRun 의 몸통을 promoteIn 으로 뽑아 둘이 쓴다.  nonce 도 한 벌.
                       두 벌이 되는 순간이 갈리는 순간이다
   주석                 한국어 godoc.  「왜」를 적는다 — 잠금이 왜 있는지 · 왜 막고 기다리는지 ·
                       왜 정산이 되돌아오는지 (CONVENTIONS 2.2)
   표기                 장식 문자 0 · glyphscan · emphasis-check
```

---

## 5. 재는 자리 — 요약

```text
   CP0    go test ./... (진짜 Postgres) · 커버리지 80% · glyphscan · vet · 크로스 빌드 · 심볼 상한 ·
          기존 라우트 회귀 (202 가 된 셋 제외 — 그것은 팩이 바꾼 값이다)
   CP2    scene-gates 3절의 명령 여섯 — 둘째 202 · QUEUED · a 끝나면 b RUNNING · 422 FAILED ·
          QUEUED 행이 requires 를 보인다(obs) · 갈래 있는 계약의 SKIPPED·chosen(obs)
   단위    4.4 의 시험 목록
```

---

## 6. 진행자에게 넘기는 것

```text
   ①  schema.sql 에 한 줄 는다 — runs_queued_idx 부분 인덱스.  obs 의 둘 뒤에 붙는다.
      접점 직렬 병합의 대상이다

   ②  대기 행에 상한이 없다 (정본).  토큰을 가진 쪽이 run_id 를 바꿔 루프로 내면 쌓인다.
      3절이 수락한 것으로 적었다 — 상한을 원하면 그것은 정책이고 constraints §2 를 여는 일이다

   ③  기존 테스트 셋이 409 에서 202 로 바뀐다 (FD logic §3.7).  「기존 라우트 15 회귀」의
      뜻이 「응답 코드가 그대로」에서 「팩이 바꾼 것 빼고 그대로」가 된다

   ④  SECURITY-08 · 12 가 obs 에서는 조건부, 여기서는 해당 없음이다.  같은 규칙에 다른 판정 —
      표면이 다르기 때문이고 2절에 근거를 적었다.  회차 표를 합칠 때 유닛별로 남긴다
```
