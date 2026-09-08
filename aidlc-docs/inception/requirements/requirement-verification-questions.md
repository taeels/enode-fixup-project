# 요구 확인 질문 — v2 시안 회차

이 회차의 산출물은 새 pen.dev 시안 1건이다. 아래 질문에 [Answer]: 뒤에
선택지 글자를 적어 달라. 선택지가 안 맞으면 마지막 Other 를 고르고 설명을
붙인다.

## Question 1
새 시안을 어디에 두나?

A) 새 파일 `design/enode-demo.pen` (권장 — "새로 하나 그린다"는 요청 그대로. 기존 `enode-ux.pen` 은 안 건드린다)

B) 기존 `design/enode-ux.pen` 에 아트보드를 이어 붙인다

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Question 2
아트보드 구성 제안. 요청을 장면 여섯으로 나눴다.

```text
   D0  온보딩 카드 시퀀스     카드 4장 + 진행 점.  애니메이션 의도는 주석으로
                             (왜 enode 인가 -> 맞춤/임대 -> 노드가 당겨 실행 -> 데모 시작)
   D1  데모 현황판 진입       Guest 로그인 버튼이 Token 입력을 대체한 첫 화면
   D2  데모 현황판 본판       웹캠 + 노드 현황판(3D 함대 뷰) + 현재 작업 목록 + 새 작업 버튼
   D3  가이드투어 오버레이     D2 위에 4 스텝 하이라이트 (웹캠 -> 노드 -> 작업 목록 -> 새 작업)
   D4  새 작업 모달           화면을 덮는 run 제출 폼
   D5  작업 그래프            3D 뷰 적용 + run 목록이 가려지지 않는 배치
```

A) 이 여섯 장 그대로

B) 온보딩 카드를 카드별 개별 아트보드로 나눈다 (D0 이 4~5장이 된다)

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Question 3
가이드투어 첫 항목이 "웹캠"이다. 기존 화면 아홉 장에는 웹캠이 없다.
데모 현황판의 웹캠은 무엇을 비추나?

A) 실물 보드 함대를 비추는 라이브 웹캠 패널 하나를 현황판에 새로 그린다

B) 노드마다 웹캠이 있어 노드 카드/상세에서 그 노드의 실물을 본다

C) 자리만 잡는 플레이스홀더로 그린다 (무엇을 비출지는 나중에)

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Question 4
함대 현황과 작업 그래프에 얹을 3D 뷰의 스타일은?

A) 기존 S6 · S7 의 로우폴리 아이소메트릭 탑뷰를 그대로 이식한다 (cc04508 시안)

B) 데모용으로 새 스타일을 만든다

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Question 5 — Security Extensions
Should security extension rules be enforced for this project?
(v1 은 켰다. v2 는 코드가 없는 시안 회차라 적용 표면이 없다 — B 권장)

A) Yes — enforce all SECURITY rules as blocking constraints

B) No — skip all SECURITY rules (이 회차 한정. 코드 회차가 다시 열면 그때 재확인)

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Question 6 — Resiliency Extensions
Should the resiliency baseline be applied to this project?
(v1 은 껐다. 시안 회차라 적용 표면이 없다 — B 권장)

A) Yes — apply the resiliency baseline as directional best practices

B) No — skip the resiliency baseline

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Question 7 — Property-Based Testing Extension
Should property-based testing (PBT) rules be enforced for this project?
(v1 은 껐다. 시안 회차라 적용 표면이 없다 — C 권장)

A) Yes — enforce all PBT rules as blocking constraints

B) Partial — pure functions and serialization round-trips only

C) No — skip all PBT rules

X) Other (please describe after [Answer]: tag below)

[Answer]:
