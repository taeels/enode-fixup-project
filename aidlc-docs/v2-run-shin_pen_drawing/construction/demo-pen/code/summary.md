# 생성 요약 — demo-pen

산출물은 `design/enode-demo.pen` 하나다. 코드는 없다.

## 만든 방법

`design/enode-ux.pen` 을 파일째 복사해 만들었다. 그래서 변수 · 컴포넌트 다섯
(Node Card · Run Row · C3 · C4 · C5) · S6 · S7 이 같은 id 로 들어왔고, 화면
여덟(S0 · S0b · S1 · S1b · S2 · S3 · S4 · S5)은 지웠다. S6 · S7 은 D2 · D5 에
필요한 부분을 복사한 뒤 지웠다. **`enode-ux.pen` 은 읽기만 했다** — 디스크
md5 가 HEAD 와 같다.

작업 중 사고 하나 — pencil MCP 의 `filePath` 는 편집기에 열린 문서가 있으면
무시된다. 첫 삭제가 열린 `enode-ux.pen` 의 메모리 문서에 갔고, 저장 전에
되돌려 디스크는 안 바뀌었다. 이후 편집기에 `enode-demo.pen` 을 열고 진행했다
(`aidlc-docs/audit.md` 2026-09-08 기록).

## 파일 안에 있는 것

```text
   컴포넌트 (원본 그대로)
   jnl0v              Node Card
   n9KdZp             Run Row                D2 · D5 의 우 열이 인스턴스로 쓴다
   pxDrainActive      C3 자원 회수 · 활성
   pxBadgeDrainBusy   C4 배지 · draining 진행 중
   pxBadgeDrainFree   C5 배지 · draining 대기 중

   아트보드 (1920 x 1080)
   CBJwz   D1 데모 진입 · Guest 로그인       y 6200
   FN7dp   D2 데모 현황판 · 함대 3D           y 7400
   Sm65l   D3 가이드투어 · 1/4 웹캠           y 8600   D2 복사 + 덮개 + 말풍선
   bnV1N   D3b 가이드투어 · 2/4 노드 현황판   x 2560   D3 복사. 스포트라이트 = 좌 패널
   JViqt   D3c 가이드투어 · 3/4 현재 작업 목록 x 4520   D3 복사. 스포트라이트 = 우 열
   Dfo8G   D3d 가이드투어 · 4/4 새 작업       x 6480   D3 복사. 스포트라이트 = 새 작업 · [시작하기]
   C9Suc   D4 새 작업 모달                    y 9800   D2 복사 + 덮개 + 카드
   Db9Rf   D5 작업 그래프 · run 3D            y 11000  D2 복사 + 좌 패널 교체

   노트 (아트보드 밖)
   ivn2L   Note D3 투어 스텝                  스텝 넷의 제목 · 본문 표
   -       Note 아트보드 밖 주석              frontend-components.md 7절
   -       Note D5 전환
```

## D2 의 구조 (D3 · D4 · D5 가 공유)

```text
   FN7dp
   +-- N4ZUG  Top Bar 72          제목 · 범위 · observed_at · 마지막 갱신 ·
   |                              뷰 전환 [격자 | 그래프 | 3D] · 게스트 배지 user_4821
   +-- IvU86  Body (padding 24 · gap 24)
       +-- FpjGV  Fleet Panel 1380 x 960 (layout none · clip)
       |   +-- M0abdq  Iso Plane      S6 의 Floor · Grid · Tile x6 · Legend ·
       |   |                          Detail Panel 을 복사해 재배치.  Panel Header 추가
       |   +-- FFLsf   Webcam Window  360 x 360 · (24, 576) · 떠 있는 창
       |                              헤더(웹캠 · LIVE) · 영상 자리 · 1.0x ·
       |                              [+] [-] [맞춤] · 우상단 크기 조절 손잡이
       +-- YR3ns  Run Column 460
           +-- Run Header · New Task Button 56 · Filter Chips x5 ·
               Run List (Run Row 인스턴스 x6) · Queued Note · Pagination Slot
```

D5 는 Fleet Panel 안의 Iso Plane 을 지우고 `Run Plane`(b9MUBp) 을 넣었다 —
S7 의 Floor · Grid 복사, 간선 둘은 새로 그린 path, 단계 셋은 S7 의 단계
프레임을 복사해 이름과 상태를 바꿨다. `flash.led` 는 모니터 모형을 빼고 S6
board-01 의 칩 모형을 넣어 보드 배정을 보인다. Detail Panel 은 S7 의 것을
복사해 값을 바꿨다. 우 열의 첫 행이 선택 상태다.

## 사양 대비 버튼 전수 대조

```text
   장    사양 (frontend-components.md)             시안의 노드
   D1    Guest 로그인                               Guest Login Button (OwJwZ)
   D2    뷰 전환 칩 격자 · 그래프 · 3D               View Switch (MSnuR) 칩 셋
         노드 모형 클릭 -> 상세 덮개 카드             Tile x6 · Detail Panel
         웹캠 확대 · 축소 · 맞춤 · 크기 조절 손잡이    Zoom Controls (Z39qW) · Resize Handle (i1694C)
         새 작업                                    New Task Button (D7FPsj)
         필터 칩 다섯                                Filter Chips (tXy82)
         Run 행 클릭                                 Run List (cFkRY) 인스턴스 여섯
   D3    다음 · 건너뛰기 (시작하기 는 노트에)          Next Button (hcWXl) · Skip Button (gADR8)
   D4    LED Toggle 작업 요청 · 사운드 재생 작업 요청  Task LED Button (zRM1A) · Task Sound Button (hELzn)
         닫기                                       Close Button (a4rz9k)
   D5    함대로                                     Back Chip (VYdMq)
         단계 클릭 · 노드 모형 클릭                  Step x3 · Detail Panel (IwtOz)
         웹캠 넷 · 새 작업 · 필터 칩 · Run 행 · 뷰 전환  D2 복사본 그대로
```

## 원본과 달리 한 것

- S6 상세 카드의 「호스트 제어판 열기」 링크를 「이 노드의 run 보기」 로 바꿨다 —
  데모 표면에는 호스트 제어판이 없고, 노드가 임대 중인 run 의 D5 로 가는 것이
  데모의 동선이다
- Copy 의 `descendants` 를 이름 키로 주면 조용히 무시된다 (id 키는 된다).
  D5 의 단계 이름 · 상태 · 범례 힌트는 복사 뒤 텍스트 노드를 찾아 id 로 고쳤다

## 값 출처

- 상태 enum 원문 — RUNNING · QUEUED · SUCCEEDED · FAILED / DONE · RUNNING · PENDING
- 색 — 원본 변수 `state-*` 그대로. 새 색은 덮개(#000000B8 · #00000099)와
  배지 반투명(기존 `#4B9CFF22` 관례)뿐
- 샘플 값 — `led-toggle-0417` 등 run_id · 시각 · `user_4821` 은 그림용이다.
  계약이 아니다 (business-rules.md R5)

## 안 한 것

- PNG 내보내기 — Build and Test
- `enode-ux.pen` · `design/exports/` · 코드 변경 없음
