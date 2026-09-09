# drain — 지켜야 하는 것

drain 유닛(W2 · CP3)의 NFR Requirements 산출물 하나다. 기술 선택은
`tech-stack-decisions.md`, 계획과 답은 `plans/drain-nfr-requirements-plan.md`.
**어긋나면 `enode-design` 이 이긴다.**

---

## 1. 답이 부른 것

물음 하나의 답이 **A** 다 — at-boundary 취소가 실패하면 Error 로그 한 줄 · 재시도 없음.
소유자의 `stop`(panel)과 다른 노드의 결과 보고가 닫는다. 새 코드 0. **위험으로 기록한다**(4.3).

## 2. security-baseline — 열다섯 규칙의 판정

새 HTTP 표면 0 · 새 파일 표면 하나(정책 파일 · 읽기만). FD rules §5 를 잇는다.

| 규칙 | 판정 | 근거 |
|---|---|---|
| SECURITY-01 암호화 | 해당 없음 (기존 코드 사실) | 저장소 새로 안 만든다. 정책 파일은 비밀이 아니다(모드 한 단어) |
| SECURITY-02 중간자 로깅 | 해당 없음 | 중간장치 없음 |
| SECURITY-03 로깅 | 준수 | 노드 로그 — 정책 값 원문(소유자의 파일) · 권한 경고 · 통보 변화 · 안 집음. Mediator 로그 — 취소 한 줄(run · node). 토큰 · principal · 계약 본문 안 찍음 |
| SECURITY-04 헤더 | 해당 없음 | HTML 없음 |
| SECURITY-05 입력 검증 | 준수 | 파일 값은 allowlist 셋으로 접는다(노드 · 저장소 두 번). `yaml.v3` 고정 타입 · 모르는 키 무시. `postResult` 갈래는 새 입력 0 |
| SECURITY-06 최소 권한 | 해당 없음 | IAM 없음 |
| SECURITY-07 네트워크 | 해당 없음 (기존 코드 사실) | 새 리스너 없음 · 광고는 기존 경로 |
| SECURITY-08 접근 제어 | 준수 | 소유 = 그 기계의 파일 쓰기(ADR-063 §3). 중앙 라우트 없음 — 토큰만으로 남의 노드를 못 뺀다. 중앙은 판정 안 함 |
| SECURITY-09 하드닝 | 준수 | 기본은 안 걸린 것. 권한 느슨하면 경고(FD Q3). 오류 문구는 로그뿐 |
| SECURITY-10 공급망 | 준수 | 새 의존 0 (`yaml.v3` 재사용) |
| SECURITY-11 설계 | 준수 | 정책 읽기는 `policy.go` 하나. 층 둘 — 파일 권한(운영자) + 어휘 검사. 오용 사례 3절 |
| SECURITY-12 인증 | 해당 없음 | 자격증명 0 |
| SECURITY-13 무결성 | 준수 | yaml 고정 타입. 취소는 `Cancel` 이라 기록(verdict · Record 봉인)이 남는다 — 누가(drain:<node_id>) 언제(ended_at) |
| SECURITY-14 경보 | 해당 없음 (범위 밖) | 운영 제외(constraints §6). **기록이지 통과가 아니다** |
| SECURITY-15 fail-safe | 준수 | 파일 오류 → 안 걸린 것(열리는 것 없음 — 오늘 그대로). 취소 실패 → 로그 · 보고는 받음 · Worker 가 창을 닫음(답 A 의 위험은 4.3) |

조건부 0 · 차단 0.

## 3. 오용 사례 (SECURITY-11)

```text
   같은 기계의 다른 사용자가 정책 파일에 at-boundary 를 써서 남의 노드를 뺀다.
   막는 것   파일 권한(0600 권장 · 느슨하면 노드 로그가 경고한다) · 그 기계에 접속했다는 것이 이미 소유 증명(ADR-063 §3)
   남는 것   회수는 파괴가 아니다 — 도는 단계는 끝까지 가고 산출은 Record 에 남는다.  소유자가 파일을 되돌리면 다음 광고에 풀린다
```

## 4. 비-보안 NFR

### 4.1 성능
```text
   정책 읽기      광고마다 파일 하나 stat+read (수 KB 아래).  광고 주기 5~60초.  캐시 없음이 옳다 — 값이 파일에 있다
   postResult     at-boundary 노드일 때만 SELECT 한 줄 + Cancel 한 tx.  아닐 때 SELECT 한 줄
   Worker 대기    at-boundary 면 광고 주기만큼 select.  CPU 0
```

### 4.2 확장
```text
   노드 여럿      기계당 설정 여럿 → 정책도 여럿(<stem>.policy.yaml).  서로 안 섞인다
   Run 여럿       여러 노드 Run 은 전체가 닫힌다 (I5).  다른 노드의 단계는 취소로 FAILED
```

### 4.3 가용 · 신뢰
```text
   수치 목표 없음
   취소 실패 (답 A)   Error 로그 · 재시도 없음.  그 Run 은 이 노드의 다음 단계에서 멈춘다 — Worker 가 안 집고
                     임대는 하트비트가 갱신하므로 회수도 안 온다.  닫는 것은 소유자의 stop(panel · CP4)이나
                     다른 노드의 결과 보고.  **수락한 위험** — 드문 DB 오류이고 stop 이 있다.  진행자에게 (6절)
   Mediator 안 닿음   광고 실패 → 정책이 안 나간다 → 중앙은 마지막 값을 본다.  통보가 늦어질 뿐이다 (decisions §2 S3 문구)
   파일 사라짐       다음 광고에 policy 가 안 실린다 = 해제.  소유자가 지운 것과 같다 — 정본이 파일이라 그것이 맞다
   경계의 창        tx 2 와 tx 3 사이에 claim 이 올 수 있다 → Worker 가 안 건다.  다른 노드는 취소로 닫힌다
   되돌림          rolled 뒤에도 취소 (FD rules §2)
```

### 4.4 유지보수
```text
   시험     internal/enode — policy_test(경로 · 없음 · 파싱 실패 · 어휘 밖 · 권한 경고 · 원인 바뀔 때만 로그) ·
            advertise 시험에 policy 실림과 응답 drain → Held · Worker 가 at-boundary 에서 안 집음 (httptest · DB 없음)
            internal/api — at-boundary 취소(단계 둘) · graceful 무취소 · rolled 뒤 취소 · 마지막 단계는 정산이 이김 · NodeDrain 오류 로그 (DB)
   커버리지  internal/api 여유 3 문장 — postResult 갈래 전부 덮는다.  internal/enode 84.7% · cmd/enode 97.2%
   한 벌     drain 어휘 상수는 internal/contract 하나.  store 가 그것을 가리킨다
   주석     왜 응답의 값인가 · 왜 Worker 가 안 집는가 · 왜 세 tx 인가
```

## 5. 재는 자리
```text
   CP0    go test ./... (진짜 Postgres) · 커버리지 80% · glyphscan · vet · 크로스 빌드
   CP3    scene-gates 3절 — 단계 둘 이상 계약 · 정책 파일에 at-boundary · GET /v1/nodes draining · 경계 뒤 lease 없음 ·
          새 계약 202 QUEUED · FAILED + verdict drain:<node_id> · 산출은 record · 풀면 대기 Run RUNNING
```

## 6. 진행자에게
```text
   ①  답 A 의 수락 위험 — 취소 실패 시 소유자의 stop 이 닫는다 (4.3)
   ②  조율 둘 — cmd/enode/main.go 한 줄(panel) · internal/enode/claim.go 한 분기(transcript)
   ③  drain 어휘 상수를 internal/contract 로 옮긴다 — store 의 상수는 별칭이 된다 (obs 파일 한 줄)
   ④  internal/api 커버리지 여유 3 문장 — 갈래를 전부 덮어야 한다
```
