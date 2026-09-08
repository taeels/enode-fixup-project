# 도메인 개념 — demo-pen

데모 화면이 다루는 개념. 셋은 API 가 이미 주는 것이고 넷은 데모가 브라우저
안에서만 갖는 것이다. **데모 전용 개념은 서버에 저장되지 않는다** — 그래야
Mediator 의 스키마와 API 를 안 건드린다.

## 1. API 가 주는 것 (그대로 쓴다)

```text
   Node     GET /v1/nodes       label · capabilities · lease? · draining ·
                                observed_at · expires_at
                                상태 여섯은 관측에서 파생 (design/README.md 2절)
   Run      GET /v1/runs        run_id · state (RUNNING · QUEUED · SUCCEEDED ·
                                FAILED) · 요약 (노드 · 단계 · 시각)
   Step     GET /v1/runs/{id}   id · state (PENDING · CLAIMED · DONE · FAILED ·
                                SKIPPED · ASKED) · node · attempt · started_at ·
                                needs[] · chosen
```

## 2. 데모가 갖는 것 (브라우저 캐시)

```text
   GuestSession    user_id        "user_" + 번호.  로그인 때 한 번 배정
                   first_visit    처음 들어왔는가.  D0 · D3 의 조건
                   tour_done      투어를 끝냈거나 건너뛰었는가
   TourStep        index 1..4 · target (webcam · fleet · runs · new-task) ·
                   title · body     글은 frontend-components.md 3절의 표
   TaskPreset      LED_TOGGLE · SOUND_PLAY
                   각각 미리 적힌 계약 하나에 대응한다.  계약 본문은 코드 회차가
                   정한다 — 시안은 버튼 이름과 부제만 갖는다
   ViewMode        fleet (D2) · run (D5)     좌 상단 패널이 무엇을 보이나
```

## 3. 관계

```text
   GuestSession 1 --- 0..1 TourStep(진행 중)
   TaskPreset   1 --- 1    Contract (코드 회차가 적는 JSON)
   ViewMode.run 1 --- 1    Run (선택된 것)
   Run          1 --- *    Step        needs[] 가 Step 사이 간선
   Step         * --- 0..1 Node        배정되면 그 노드의 모형이 단계 자리에 선다
```

## 4. 안 만드는 것

- 사용자 계정 · 비밀번호 · 권한. user_<번호> 는 식별이지 인증이 아니다
- 이름 사전(형용사 + 이름). Q2 로 뺐다
- 계약 편집기. Q1 로 뺐다
