# 공개 데모 제출 계약 — ui · demo-back 공동 검토안

**상태**: 담당 설계안. 경로·입력·오류·재시도 규칙을 이 문서 한 곳에서 정의한다.
현재 서버에 이 라우트가 있다는 뜻은 아니다. 기준 구현은 `0a159a4`다.

## 1. 요청

`POST /v1/demo/runs`, `Content-Type: application/json`.
`Config.Demo=true`에서만 등록한다. 브라우저는 Authorization·principal을 보내지 않는다.

```json
{
  "scenario_id": "welcome-audio",
  "submitter": "guest-bright-otter",
  "request_id": "04995a39-1fba-4c16-b2cd-dc80ff21257e"
}
```

예시는 합성이다. 서버는 세 문자열만 허용하고, 임의 계약·run_id·requires·steps·
토큰·파일 경로·임의 이름 입력 옵션을 받지 않는다. JSON 객체 하나, 중복/알 수 없는
키 없음, 뒤따르는 JSON 없음, 본문 최대 4 KiB다. 실제 Content-Type을 검사한다.

| 필드 | 규칙 |
|---|---|
| scenario_id | 정확히 `led-toggle` 또는 `welcome-audio`. 진행자 example 이름과 분리한 공개 별칭 |
| submitter | 기존 guest.guestName()의 값. `^guest-[a-z]{1,24}-[a-z]{1,24}$`, 최대 55 ASCII 바이트. 인증 신원으로 사용하지 않음 |
| request_id | 소문자 UUID v4, 36자. 새 작업 의도마다 브라우저 crypto.randomUUID() 또는 getRandomValues로 생성 |

기존 Guest 생성 결과는 위 이름 규칙을 만족한다. 저장된 이름이 훼손된 경우에는
새 작업을 보내지 않고 기존 이름 생성기로 재발급하는 복구를 제공한다.
이름 사전이나 두 번째 이름 입력 폼은 추가하지 않는다.

## 2. 재시도와 Run 식별

서버는 `[scenario_id, request_id, submitter]`를 공백 없는 JSON 문자열 배열로
직렬화한 UTF-8의 SHA-256 전체 소문자 hex를 구하고, `demo-`를 앞에 붙여 run_id로
쓴다. 이 정규 표현을 ui에 복제하지 않는다. submit 응답의 run_id를 소비한다.

같은 요청은 같은 Run을 조회·접수한다. 이름이나 시나리오가 다른 요청은 같은
request_id를 실수로 사용해도 다른 Run이다. 새 작업 버튼의 새 의도는 새 UUID다.
토큰·시각·현재 노드 상태를 식별 계산에 넣지 않는다. 서버 재시작에도 같은 결과다.
새 멱등성 테이블은 만들지 않고 기존 submit의 run_id 중복 처리·트랜잭션을 사용한다.

브라우저는 진행 중 요청의 세 값과 시작 시각을 sessionStorage에 보관한다.
10초 안에 확정 응답이 없으면 '접수 결과를 확인하지 못했습니다'를 표시하고 같은
요청의 재시도만 제공한다. 자동 POST 재전송은 하지 않는다. 새로고침 뒤에도 기존
요청을 복구한다. 저장이 막혀 있으면 메모리에서 같은 탭의 재시도를 유지하고,
페이지 이동 시 미확인 요청을 잃는다는 안내를 한다.

## 3. 응답

기존 submit의 상태 코드와 runView/오류 본문을 그대로 전달한다. 새로운 성공
포장 객체를 만들지 않는다. 접수 직후에는 requires·submitter가 없을 수 있으므로
GET detail·목록을 다시 읽는다. 목록에 없는 접수 결과를 실제 관측 행으로 만들지 않는다.

| 결과 | UI 행동 |
|---|---|
| 201 + run_id/state | 모달 닫기, 그 Run 선택, 3D 작업 그래프, 목록·상세 갱신 |
| 202 + run_id/state=QUEUED | 접수됨·대기 중 표시, 같은 선택 전환. 배정 노드·큐 순번을 만들지 않음 |
| 200 + 기존 Run | 같은 요청의 기존 결과로 이동. state가 FAILED면 실패 상태를 그대로 표시 |
| 400 | 잘못된 필드/형식/임의 계약/미등록 시나리오. 같은 잘못된 요청 자동 재시도 없음 |
| 413 / 415 | 본문 초과 / Content-Type 불일치. 서버가 접수하지 않음 |
| 404 | 데모 라우트 미등록. '작업 제출을 사용할 수 없습니다' 표시 |
| 409 | 현재 기존 접수의 배정 충돌. queue 인수 후 의미 재대조. QUEUED로 꾸미지 않음 |
| 422 | 기존 계약/능력 거절. 이유 표시·Run 목록 갱신. 성공 안내나 실행 그래프 전환 없음 |
| 429 | 한도 초과. Retry-After 이전 재시도 금지, 요청 ID 유지 |
| 503 / 네트워크·timeout / 성공 코드인데 잘못된 본문 | 접수 결과 미확인. 같은 요청으로 확인·재시도 |

201/202라도 run_id/state가 없으면 접수 확정을 하지 않는다. 요청 취소/모달 닫기를
서버 작업 취소로 표시하지 않는다. 4xx에서 Run 행이 생길 수 있는 기존 동작은
그대로이며, 후속 200 재접수가 FAILED를 반환해도 성공으로 고치지 않는다.

오류는 기존 `{"error":{"code":400,"reason":"invalid demo request"}}` 형식을 쓴다.
신규 reason은 영어 고정 문구, UI 안내는 한국어다. token·원문 계약·내부 경로를
오류에 넣지 않는다. token이 비어 있으면 503으로 제출을 막고 인증 검사를 우회하지 않는다.

## 4. 시나리오 연결의 선행 조건

공개 별칭은 임의 `contract.Example(name)` 호출로 전달하지 않는다. 별도 고정
매핑의 승인된 example과 이름 주입 함수만 실행한다. 현재 examples에는
agent·command·multi만 있고, 실제 LED/음원 예제는 확인하지 못했다.

진행자가 준 파일에서 requires·steps·needs·success_when·음원 blob 경로를 검토한다.
LED의 heartbeat/persistent-on 두 계약을 한 버튼에 어떻게 연결할지는 픽스처를
받아 확정한다. 서버 메모리의 번갈아 선택이나 추측한 하드웨어 명령은 만들지 않는다.
픽스처/주입 검증이 안 된 항목은 503 `demo scenario is not configured`로 막는다.
