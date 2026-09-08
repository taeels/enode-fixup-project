# ui 소비 API 계약표 — obs 인수 반영

**기준**: 구현 `0a159a4`, `enode-design` gitlink `29c89cd`, 2026-09-09 KST.
경로는 저장소 루트 기준이다. 병합된 obs 코드와 기존 관측 테스트를 대조했다.
이 문서는 UI가 소비할 계약을 기록하며 서버 계약을 새로 만들지 않는다.

## 1. 근거와 상태

| 표기 | 뜻 | 근거 |
|---|---|---|
| 정본 | 문서가 요구하는 형태·의미 | `enode-design/protocol/mediator-api.md`, ADR-065·069·060, `requirements/` |
| 현재 | 이 체크아웃에서 확인한 직렬화·오류 | `internal/api/api.go`, `internal/store/observe.go`, `internal/store/ask.go`, `internal/contract/contract.go` |
| 합성 | 테스트 전용 입력. 실제 함대 캡처가 아님 | 오류 입력과 queue 후속 시나리오를 구분 |

불변식은 `requirements/canon.md` 에 따라 `protocol/INVARIANTS.md` 가 우선한다.
현재 코드와 요구가 다르면 차이를 기록하고, UI 편의를 위해 서버의 새 의미나
양쪽 형태를 무조건 받아들이는 파서를 먼저 만들지 않는다.

## 2. 요청 경계

| 요청 | 목적 | 인증·필터 | 현재 상태 |
|---|---|---|---|
| `GET /ui/` | 랜딩·토큰 진입 | 무인증 정적 파일 | 기존 landing 과 guest 경로가 있음 |
| `GET /v1/nodes` | 함대 스냅샷 | 실 모드 Bearer, 데모 무인증. 필터 없음 | obs 병합됨 |
| `GET /v1/runs` | Run 목록 | 같은 모드 규칙. `state`, `since`, `work`, `limit` | obs 병합됨 |
| `GET /v1/runs/{id}` | 선택한 작업·배정된 Run 의 단계 | 같은 모드 규칙. ID는 경로 세그먼트로 인코딩 | requires·chosen 확장됨 |
| `GET /v1/asks` | CP7 사람 대기 표시 | 같은 인증 | 현재 존재. `{asks: [...]}` 이며 observed_at 없음 |

실 함대 UI 는 읽기만 한다. 중앙 drain·submit·answer 조작을 추가하지 않는다.
공개 데모는 `Config.Demo` / `ENODE_DEMO_MODE=1`일 때 nodes·runs·detail 셋을
토큰 없이 읽는다. asks는 계속 인증이 필요하므로 공개 데모에서 조회하지 않는다.
제출은 demo-back 경로를 사용한다. 실 함대 토큰을 guest 경로에 전달하지 않는다.

## 3. Nodes 스냅샷

근거: `mediator-api.md` GET /v1/nodes, ADR-065, `enode-features.md` §3.1.1.

| 필드 | 형태·의미 | UI 처리 |
|---|---|---|
| `observed_at` | 스냅샷의 시각 문자열 | 성공 시각과 함께 보존. 브라우저 시계와 구분 |
| `nodes` | 배열. 만료되지 않은 광고만 | 이 배열로 카드 구성. 만료됐다고 단정해 로컬에서 삭제하지 않음 |
| `node_id`, `label`, `instance` | 문자열 | node_id 로 결합. label 은 표시값, instance 는 재기동 구별 |
| `capabilities` | `[{capability, attrs: {key: string}}]` | 모르는 속성 키도 텍스트로 표시 |
| `seen_at`, `expires_at` | 광고 시각 | 잔여 30초 이하이면 흐리게 표시. 프로세스 생존 확정으로 쓰지 않음 |
| `lease` | 항상 키가 있음. `{run_id, not_after}` 또는 `null` | null은 관측 임대 없음. 생략·빈 객체는 계약 불일치 |
| `draining` | `""`, `"graceful"`, `"at-boundary"` | boolean 이 아님. 모드와 관측된 임대 유무를 분리 표시 |

`principal` 은 이 응답에 없다. `free=true` 필터도 없다. lease 부재는 그 순간
임대가 관측되지 않았다는 의미이며, 배정 가능 판정이 아니다.
`ASKED` 동안에는 not_after 가 지났어도 사람 대기 상태를 유지할 수 있다(ADR-047).

## 4. Run 목록과 상세

### 목록 — 정본 및 확장 요구

`{observed_at, runs: [...]}`. `created_at` 내림차순. limit 기본 100이며
페이지네이션은 없다. UI 가 임의의 서버 필터·커서를 추가하지 않는다.

| 필드 | 계약·주의점 |
|---|---|
| `run_id`, `state`, `work_id`, `created_at` | 목록 식별·정렬·표시. 서버의 상태 어휘를 보존 |
| `ended_at`, `verdict` | 종료 전 null. 종료 verdict 구체 형태는 D3 확인 |
| `assigned` | `[{as, nodes: [{node, label}]}]`. QUEUED 는 빈 배열이라는 정본 예시 |
| `submitter` | 항상 문자열. 기존/미지정 Run은 빈 문자열. 상세에는 없으므로 목록에서 결합 |

목록에 `steps`·`requires` 를 붙이도록 요구하지 않는다. QUEUED 행을 선택하면
상세를 읽어 요구 능력을 표시하고 어떤 노드 카드에도 얹지 않는다.

### 상세 — 현재 코드 + obs 확장

| 필드 | 근거·현재 모습 | UI 계약 |
|---|---|---|
| `run_id`, `state` | 현재 `runView` 의 필수 키 | 선택 요청의 Run ID 와 일치해야 함 |
| `assigned`, `steps` | 현재 `omitempty` | 생략을 처리하되, RUNNING 의 steps 부재를 완료/성공으로 해석하지 않음 |
| `requires` | 현재 유효한 계약의 요구, `attrs`는 중첩 문자열 맵 | `as`로 steps[].uses 연결. 읽기 실패의 200+warnings+생략은 부분 응답 |
| `steps[].seq`, `id`, `state`, `uses` | 현재 StepView | id 가 그래프 노드, seq 가 순서·되묻기 연결 키 |
| `steps[].needs` | 현재 항상 배열, 시작점은 `[]` | 이 값으로 간선 작성. 계약의 기본 needs 를 브라우저가 재계산하지 않음 |
| `steps[].node` | 현재 생략 가능 | assigned[].nodes[].node 및 node_id 와 연결 |
| `steps[].attempt`, `started_at`, `ended_at` | 현재 생략 가능 | 없는 시각·회차를 만들어 표시하지 않음 |
| `steps[].chosen` | 항상 boolean 키가 있음 | 누락은 계약 불일치. SKIPPED+true 와 SKIPPED+false 를 구별 |

`as` 라는 최상위 필드를 요구하지 않는다. `requires[].as` 가 역할 별칭이다.
`chosen=false` 는 실패나 대기를 뜻하지 않는다. `state` 와 독립된 사실이다.
상세의 `verdict` 는 현재 `store.Verdict{state, checks, fleet?}` 객체이므로 목록과
같은 원시 문자열이라고 가정하지 않는다.

## 5. 오류와 연결 상태

현재 `fail()` 의 JSON 형태는 `{"error":{"code":401,"reason":"missing or invalid token"}}`
다. 상세 미존재는 404 `no such run`, 조회 실패는 503 `query failed`.
병합된 obs도 이 공통 오류 형식을 사용한다. 상세의 requires 읽기 실패는
HTTP 200과 warnings로 보고되므로, Run 상태를 유지하면서 요구 정보 미확인을
보여준다. 이를 정상 빈 requires나 완전한 갱신으로 처리하지 않는다.

- 401 은 인증 상태로 즉시 전환하고 보호된 화면을 숨긴다. 네트워크 재시도 셋을
  기다려 인증 실패를 흐리지 않는다. 새 토큰 제출 전 반복 요청을 멈춘다.
- 404 상세는 선택한 Run 을 찾을 수 없다는 표시다. 함대 전체 실패와 구별한다.
- 공개 읽기의 429는 한도 초과다. `Retry-After: 1`을 존중해 그 시각 전에는
  공개 조회 셋의 자동/수동 재요청을 하지 않는다. 토큰 입력으로 전환하지 않는다.
- 읽기 셋의 성공 응답은 `Cache-Control: no-store`다. nodes에 `?t=` 같은
  질의 인자를 붙이면 400이므로 캐시 회피 인자를 만들지 않는다.
- 503·네트워크 실패·200인데 JSON/필수 형태가 잘못된 응답은 성공으로 세지 않는다.
- 5초 간격, 연속 3회 실패 시 S1b. 지연된 요청이 다음 주기를 막지 않도록 FD 에서
  timeout 을 명시한다. 초안 제안은 5초 이내 요청 종료와 주기당 중복 요청 방지다.
- nodes/runs 두 요청은 하나의 DB 스냅샷이 아니다. 관측 시각을 보존하고 순간적인
  행 불일치를 가능한 관측 차이로 다룬다. 마지막 성공 데이터를 샘플로 교체하지 않는다.

## 6. obs 인수 결과 — 2026-09-09

| ID | 확인 사항 | 근거와 준비 상태 | 확인 주체·시점 |
|---|---|---|---|
| D1 | requires 속성 | 관측은 중첩 attrs, 입력 계약은 평탄. 정상 소비 계약 확정 | 0a159a4 코드·기존 관측 테스트 통과 |
| D2 | lease 부재 | 키가 항상 있고 null. 생략 후보는 오류 입력으로 보존 | 같은 기준 |
| D3 | 목록 verdict | Verdict 객체 또는 null, checks[].note 보존 | 같은 기준. 실제 drain 장면은 후속 |
| D4 | submitter 과거값 | 목록 문자열, 미지정은 빈 문자열. 기존 저장 연결도 구현됨 | 같은 기준. Guest 검증·queue 전달은 후속 |
| D5 | requires·chosen | chosen 항상 boolean. requires는 정상 조회에 있고 읽기 실패 시 warnings+생략 | 같은 기준. QUEUED 실연동은 후속 |
| D6 | sandbox 표시 출처 | 현재 contract·Advert 타입에 전용 필드 없음. OS 종류·하네스명에서 추론하지 않음 | ui FD. 미확정이면 데모 완료 조건을 닫지 않음 |
| D7 | 실 모드 진입 경로와 guest 읽기 경계 | 공개 읽기 셋은 Config.Demo로 구현됨. 기존 /ui/ 마운트 재사용 | UI 진입·제출 계약은 FD 보완 |

검증 환경·근거·남은 장면은 [obs 인수 기록](obs-integration-review.md)에 있다.
2026-09-09의 계약표가 이전 후보 설명보다 우선한다.

사용자가 이미 위임한 구현 선택은 코드·정본을 더 읽어 정한다. 다른 담당의
계약이 필요한 항목은 검토 자료로 남긴다. 미정을 답변 완료나 구현 승인으로
표시하지 않는다. 불일치를 발견하면 같은 원인을 둔 두 파서를 영구 유지하기보다
합의한 응답 계약 하나에 맞춘다.
