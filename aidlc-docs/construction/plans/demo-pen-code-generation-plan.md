# Code Generation 계획 — demo-pen

이 문서가 Code Generation 의 유일한 정본이다. Part 2 는 아래 단계를 순서대로
실행하고 끝날 때마다 [x] 를 찍는다.

## 유닛 맥락

- **산출물**: `design/enode-demo.pen` — 아트보드 D1~D5 (1920 x 1080)
- **사양**: `aidlc-docs/construction/demo-pen/functional-design/frontend-components.md`
  (버튼 전수 · 배치 치수 · 전환 표 · 아트보드 밖 주석)
- **규칙**: 같은 폴더의 `business-rules.md` R1~R7
- **의존**: 없음. 코드 · API 를 안 건드린다
- **도구**: pencil MCP `execute` 에 `filePath` 로 새 파일을 지정한다 (편집기에
  열린 파일이 아니어도 동작함을 2026-09-08 에 확인). `.pen` 은 Read/Grep 하지 않는다

## 소재 (enode-ux.pen 에서 가져오는 것)

```text
   변수         bg · surface · surface-2 · border · border-strong · text-* ·
                state-idle/leased/drain/queued/stopped/expiring · font-body(Inter) ·
                font-mono(JetBrains Mono) · gap-sm/md/lg · radius
   컴포넌트      jnl0v Node Card · n9KdZp Run Row · pxDrainActive C3 ·
                pxBadgeDrainBusy C4 · pxBadgeDrainFree C5
   S6 WG8zm     Iso Plane MZxBs — Floor · Floor Grid · Tile x6 (ws-a/b/c/d ·
                board-01/02) · Legend Row · Detail Panel(rY4jl)
   S7 NCouv     Iso Plane g9Pl2O — Floor · Floor Grid · Edge x5 · Step x5 ·
                Legend Row · Detail Panel(D4yaX)
   S0 cQxam     가운데 카드 (D1 의 틀)
```

## 방법

`enode-ux.pen` 을 **파일째 복사**해 `enode-demo.pen` 을 만든다. 그러면 변수 ·
컴포넌트 · S6 · S7 이 같은 id 로 들어 있어 cross-file 복사가 필요 없다.
필요 없는 화면은 지우고, S6 · S7 은 소재로 두었다가 D2 · D5 를 다 만든 뒤
지운다. **`enode-ux.pen` 은 읽기만 한다** (Q1 · R6).

## 단계

- [ ] Step 1. 파일 만들기 — `cp design/enode-ux.pen design/enode-demo.pen`.
      S0 · S0b · S1 · S1b · S2 · S3 · S4 · S5 프레임 여덟을 지운다.
      남는 것: 컴포넌트 다섯 · S6 · S7. `git status` 로 `enode-ux.pen` 불변 확인
- [ ] Step 2. D1 데모 진입 — 1920 x 1080 프레임. S0 의 카드 구성을 본떠 새로
      그린다 (제목 · 설명 · Guest 로그인 버튼 · 각주). placeholder 로 시작해 끝나면 해제
- [ ] Step 3. D2 본판 뼈대 — 1920 x 1080 프레임 · 상단바(제목 · observed_at ·
      마지막 갱신 · 뷰 전환 칩 격자/그래프/3D · 게스트 배지 user_4821) ·
      좌 패널 1380 x 960 · 우 열 460
- [ ] Step 4. D2 좌 패널 — S6 Iso Plane 의 Floor · Floor Grid · Tile 여섯 ·
      Legend Row · Detail Panel 을 복사해 1380 x 960 에 맞게 배치한다. 모형은 웹캠
      기본 자리(좌 하단 360 x 360)를 피한다. Detail Panel 은 우상단 덮개 카드
- [ ] Step 5. D2 웹캠 창 — 360 x 360 떠 있는 창(좌 하단 · 안쪽 여백 24 · z 위).
      헤더(웹캠 · LIVE 점) · 플레이스홀더 본문 · 우하단 컨트롤 [+][-][맞춤] · 배율
      1.0x · 우상단 크기 조절 손잡이
- [ ] Step 6. D2 우 열 — 헤더 · [새 작업] · 필터 칩 다섯 · Run Row 인스턴스 여섯
      (RUNNING 2 · QUEUED 2 · SUCCEEDED 1 · FAILED 1, run_id 는 led-toggle-* ·
      sound-play-*) · 안내 상자 · 페이지네이션 자리. D2 검증 · placeholder 해제
- [ ] Step 7. D3 투어 — D2 를 Copy. 덮개(검정 72%) 를 우 열 포함 전체에 얹되 웹캠
      창 영역은 뚫는다(덮개를 네 조각으로). 말풍선(1 / 4 · 제목 · 본문 · 건너뛰기 ·
      다음). 아트보드 밖에 스텝 넷의 표 노트
- [ ] Step 8. D4 모달 — D2 를 Copy. 덮개(검정 60%) · 가운데 카드 720 (제목 · 닫기 ·
      설명 · 버튼 둘 + 부제 · 각주)
- [ ] Step 9. D5 run 뷰 — D2 를 Copy. 좌 패널 내용을 S7 Iso Plane(Floor · Grid ·
      Edge 5 · Step 5 · Legend · Detail Panel)으로 바꾼다. 헤더를 [<- 함대로] ·
      함대 / led-toggle-0417 · RUNNING 으로. 단계 이름을 plan · flash.led · verify
      셋으로 줄이고 flash.led 를 RUNNING · board-01 배정으로. 우 열의 첫 행을
      선택 상태로. 웹캠 창 그대로
- [ ] Step 10. 아트보드 밖 주석 — frontend-components.md 7절의 다섯 줄 + 웹캠 큰
      상태 한 줄을 note 노드로. D0 순서 · 캐시 · S6/S7 복사 · 폴링 · 웹캠
- [ ] Step 11. 정리 — 소재 S6 · S7 프레임 삭제. 모든 placeholder 해제.
      `Get` 으로 clipped 문제 0 확인. D1~D5 스크린샷으로 눈 검사
- [ ] Step 12. 문서 — `aidlc-docs/construction/demo-pen/code/summary.md`
      (만든 노드 · 소재 출처 · 사양 대비 버튼 전수 대조표)

## 사양 추적 (버튼 전수)

```text
   D1   Guest 로그인
   D2   뷰 전환 칩 3 · 노드 모형 클릭 · 웹캠 확대/축소/맞춤/손잡이 · 새 작업 ·
        필터 칩 5 · Run 행 클릭
   D3   다음 · 건너뛰기 (· 시작하기 는 주석 표에)
   D4   LED Toggle 작업 요청 · 사운드 재생 작업 요청 · 닫기
   D5   함대로 · 단계 클릭 · 노드 모형 클릭 · 웹캠 4 · 새 작업 · 필터 칩 5 ·
        Run 행 클릭 · 뷰 전환 칩 3
```

## 안 하는 것

- `enode-ux.pen` · `design/exports/` 변경
- PNG 내보내기 — Build and Test 가 한다
- 코드 · README 변경
