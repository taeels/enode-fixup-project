# Functional Design 계획 — demo-pen

유닛 이름은 `demo-pen` 하나다 (Units Generation 을 건너뛰었으므로 이 회차의
산출물 전체가 한 유닛이다). 산출물은 아트보드 D1~D5 의 화면 사양이고, 그것을
Code Generation 이 `design/enode-demo.pen` 으로 그린다.

## 근거로 읽은 것

- `aidlc-docs/inception/requirements/requirements.md` — D1~D5 와 버튼 목록
- `design/README.md` — 표면 둘의 책임 · 노드 카드 여섯 상태 · 어휘 규칙 (6절)
- `design/exports/S0-token.png` · `S1-fleet-grid.png` · `S2-run-graph.png` —
  기존 배치. 폭 1440 (2x export 2880). S1 은 좌 격자 + 우 Run 목록, S2 는
  좌 그래프 + 우 선택 단계 패널
- S6 · S7 은 export 가 없다. cc04508 커밋 메시지가 사양이다 — 아이소메트릭
  평면 · 로우폴리 컴퓨터 모형(워크스테이션은 모니터+타워, 보드는 칩) ·
  작업 중인 노드만 상태색으로 화면이 켜짐 · 런 뷰는 배정 단계에 모형, 미배정
  단계에 슬롯 마커 · 상단바에 격자/그래프/3D 전환 칩
- `internal/contract/contract.go` — `Contract{run_id, work, requires[], lease,
  steps[], success_when[], ledger?}`. 내장 예시 셋: `agent` · `command` · `multi`
  (`internal/contract/examples/`)
- `POST /v1/runs` 응답 — `201 runView{run_id, state, assigned?, reject?, ...}`
  거절은 `409 CodeAllBusy` · `422 CodeNoCandidate`

## 계획

- [ ] 1. 화면 흐름과 상태 전이를 닫는다 (D1 -> D2 -> D3 -> D4 -> D5)
- [ ] 2. 아트보드마다 영역 배치 · 요소 · 버튼 전수 · 전환을 적는다
- [ ] 3. 데모 전용 도메인 개념을 정의한다 (Guest 세션 · 이름 · 투어 진행 · 첫 방문)
- [ ] 4. 규칙을 적는다 — 어휘(enum 원문) · 값을 지어내지 않기 · 표기 규약
- [ ] 5. 산출물 넷을 낸다
  - `aidlc-docs/construction/demo-pen/functional-design/frontend-components.md` (주 문서 — 화면 사양)
  - `aidlc-docs/construction/demo-pen/functional-design/domain-entities.md`
  - `aidlc-docs/construction/demo-pen/functional-design/business-rules.md`
  - `aidlc-docs/construction/demo-pen/functional-design/business-logic-model.md`
- [ ] 6. 승인 받고 커밋

## Code Generation 의 전제 (지금 알려둔다)

pencil MCP 가 **pen.dev 편집기에 파일이 열려 있어야** 동작한다. 지금은
"A file needs to be open in the editor" 로 막힌다. Functional Design 은 export
PNG 로 충분하지만, Code Generation 에 들어가기 전에 pen.dev 앱에서
`design/enode-ux.pen` 을 열어 두어야 한다 (S6 · S7 모형을 복사해 오기 위해).

## 질문

아래 [Answer]: 뒤에 글자를 적어 달라. 여섯 다 시안의 모양을 가르는 것이다.

### Question 1
D4 새 작업 모달의 폼은 어떤 모양인가?

A) 내장 예시 셋(agent · command · multi) 중 하나를 카드로 고르고, 오른쪽에 그 계약 JSON 이 편집 가능한 상태로 보인다 (권장 — 데모 관람자가 계약 문법을 몰라도 제출할 수 있고, 값을 지어내지 않는다)

B) 계약 JSON 편집기 하나만 있다 (runctl submit 과 같은 입력)

C) 필드별 폼 — run_id · work · requires · steps 를 각각 입력 칸으로 푼다

X) Other (please describe after [Answer]: tag below)

[Answer]:

### Question 2
D1 Guest 로그인의 이름은 언제 보이나?

A) D1 에서 버튼을 누르면 이름이 뽑히고, 그 이름 배지를 단 채 D2 로 넘어간다 (D1 은 버튼과 설명만. 권장)

B) D1 에 뽑힌 이름이 미리 보이고 "이 이름으로 입장" 을 누른다 (다시 뽑기 는 그리지 않는다)

X) Other (please describe after [Answer]: tag below)

[Answer]:

### Question 3
D2 데모 현황판의 영역 배치는?

```text
   A                                    B
   +----------+-----------------------+  +----------------------+----------+
   |  웹캠     |  3D 함대 탑뷰          |  |  3D 함대 탑뷰         |  현재    |
   |  (좌상)   |  (우상, 크게)          |  |  (좌, 크게)           |  작업    |
   +----------+-----------------------+  |                      |  목록    |
   |  현재 작업 목록 (하단 가로)         |  +----------------------+  (우측)  |
   |                        [새 작업]  |  |  웹캠 (좌하, 가로)     | [새 작업]|
   +----------------------------------+  +----------------------+----------+
```

A) 웹캠 좌상 · 3D 함대 우상 · 현재 작업 목록 하단 가로

B) 3D 함대 좌상 크게 · 웹캠 좌하 · 현재 작업 목록 우측 세로 (S1 의 우측 목록 자리를 지킨다. 권장)

X) Other (please describe after [Answer]: tag below)

[Answer]:

### Question 4
D5 작업 그래프에서 run 목록이 안 가리게 하는 배치는?

A) 좌 3D 그래프 · 우 run 목록 세로 (S1 의 우측 목록이 그대로 남고, 선택 단계 상세는 그래프 위 작은 패널로. 권장)

B) 상 3D 그래프 · 하 run 목록 가로 (S2 의 우측 패널은 그대로 두고 아래에 목록을 더한다)

C) 세 열 — 좌 3D 그래프 · 중 선택 단계 상세 · 우 run 목록

X) Other (please describe after [Answer]: tag below)

[Answer]:

### Question 5
D3 가이드투어의 표현 방식은?

A) 스포트라이트 — 화면을 어둡게 덮고 대상 영역만 밝히며 옆에 말풍선 (1/4 진행 표시 · 다음 · 건너뛰기. 권장)

B) 번호 마커 — 대상 넷에 1~4 마커를 동시에 찍고 우측에 설명 카드 하나

X) Other (please describe after [Answer]: tag below)

[Answer]:

### Question 6
아트보드 폭은?

A) 1440 (기존 S0~S7 과 같다. 권장)

B) 1920 (공개 데모 전시용 큰 화면)

X) Other (please describe after [Answer]: tag below)

[Answer]:
