# 유닛-게이트-기능 매핑 (스토리 맵) — Units Generation

AI-DLC Units Generation 산출물이다. 여덟 유닛이 어느 **장면 게이트**(CP0~CP11)를
재고 어느 **기능**(일곱 + 시연 다섯)을 지는지 잇는다. 유닛 정의는
`unit-of-work.md`, 의존·파일 행렬은 `unit-of-work-dependency.md`.

이 저장소는 User Stories 를 건너뛰었다 — 팩의 장면(`scene-gates.md`)과 수용
기준(CP0~CP11)이 여정이자 스토리다(execution-plan). 그래서 스토리 맵은 유닛을
그 게이트·기능에 잇는 표다.

---

## 1. 게이트 배정 (CP0~CP11)

```text
   게이트   소유 유닛           협력                          완결성
   ──────   ─────────────────   ───────────────────────────   ─────────────
   CP0      (없음 · 회귀)        모든 유닛이 착수·병합 전 재실행  기동이 안 깨졌다
   CP1      obs + ui            —                             보인다 (API + 화면)
   CP2      queue + ui          obs                           기다린다 (QUEUED)
   CP3      drain               queue                         돌려받는다
   CP4      panel               obs·queue·drain·ui            제어판 (옛 정본)
   CP5      mcp                 obs                            감싼다
   CP6      transcript          panel·obs                     보인다 (하네스)
   CP7      ui(표시) + mcp(도구)  obs                           물어본다
   CP8      ui                  obs [온보딩 카드 = 외부]        들어온다
   CP9      ui + demo-back      obs·queue                     낸다
   CP10     ui + demo-back      + 하드웨어(rpi·mac·웹캠)         한 장면 (가치 정본)
   CP11     ui                  —                             떠 있다 (웹캠)
```

**CP0 은 단일 소유자가 없다** — 모든 유닛이 착수·병합 전 재실행하는 회귀
게이트다(기존 테스트·커버리지·glyphscan·포맷·vet·크로스 빌드·심볼 상한 · 기존
15 라우트). 구조 불변식(임포트 금지 넷)은 CP0 이 아니라 panel 유닛이 낸다.

**CP4 와 CP10** — 전환 전 완결 정본은 CP4(dhseo 장면·제어판)였고, **대회 데모
전환으로 정본은 CP10(데모 완주)** 이다(decisions §8 · scene-gates §5). CP4 는
전환 전 기록으로 남되 panel 유닛이 그대로 진다.

**공동 게이트 둘** — CP7 은 ui(현황판 카드 표시)와 mcp(도구 asks.list·run.answer)가
나눠 지고 각 완료 조건에 나눠 박는다(Q1=A). CP9 는 ui(모달·전환·카드)와
demo-back(allow-list·submitter 쓰기)이 나눠 진다.

---

## 2. 기능 → 유닛 (누락 0)

기능 일곱(3.1~3.3)과 시연 다섯(3.4)이 전부 배정됐다.

```text
   기능                        유닛                    게이트
   ─────────────────────────   ─────────────────────   ──────
   3.1.1 중앙 현황판            obs(API) + ui(화면)      CP1·CP2
   3.1.2 호스트 제어판          panel                    CP4
   3.1.3 하네스 트랜스크립트     transcript               CP6
   3.2.1 대기열                 queue                    CP2
   3.2.2 소유자 자원 제어(drain) drain                    CP3
   3.2.3 되묻기                 ui(표시) + mcp(도구)      CP7
   3.3.1 Mediator MCP           mcp                      CP5
   3.4.1 온보딩                 ui(가이드투어)            CP8
                               [온보딩 카드 = 외부 브랜치]
   3.4.2 게스트 데모 대시보드    ui                       CP8
   3.4.3 고정 시나리오 제출      demo-back(서버) + ui(모달·전환)  CP9
   3.4.4 제출자 이름            demo-back(쓰기) + obs(컬럼·목록) + ui(카드)  CP9
   3.4.5 웹캠 플로팅            ui                       CP11
```

**온보딩 카드 시퀀스**(3.4.1 「왜 enode 인가」 애니메이션)는 이 분해 밖 — 별도
브랜치 작업이고 pen 내용이 아니다(2026-09-08 사용자 결정). ui 유닛의 3.4.1 지분은
대시보드 가이드투어다.

**sandbox 표시**(3.4.2 의 노드 강조 셋 capability·label·sandbox 중 하나)는 ui 의
노드 표현에 자리를 두되 출처는 FD(decisions §8.5). capability·label 은 이미 있다.

---

## 3. 화면 검증 — 유닛별 조작 나열 (scene-gates §2.1·§5.3)

화면이 있는 유닛의 완료 조건은 **그 화면의 버튼·조작을 전부 나열**한다. 「누르면
걸린다」만으로는 「풀기」가 빠진 채 통과된다.

```text
   ui · 실 함대 모드
     S0    토큰 입력 · S0b 401 · 탭 재방문 재요청(세션 저장소)
     S1    카드 노드 수만큼 · 임대 not_after(진행 막대 아님) · draining 배지
           두 국면(진행 중 C4 · 대기 중 C5) · 마지막 갱신 시각
     S1b   폴링 끊김 — 카운트다운 멎고 실패 표시
     S2    작업 그래프 · QUEUED 행 어느 카드에도 안 얹힘 · 안 간 갈래 SKIPPED·chosen
   ui · 공개 데모 모드
     게스트  Guest 로그인 → 랜덤 2단어 이름(= submitter · 토큰 입력 아님)
     투어    가이드투어 4스텝(웹캠·노드·목록·새작업)
     모달    새 작업 → 고정 시나리오 둘(이름 칸 없음 · 임의 계약 없음)
     전환    작업 그래프(3D S7)로 전환 · run 목록 안 가림 · RUN 카드에 submitter
     웹캠    좌하단 floating · resize · 레이아웃 불변(CP11)
     3D      함대 3D(S6) · 작업 그래프 3D(S7) · 격자/그래프/3D 뷰 전환
   ui · 되묻기 (CP7 표시)
     S1    「사람을 기다림」 여섯째 상태 · 카운트다운 not_after 아님 · answerers·칠 명령
   panel (CP4)
     S3    조작 넷 status·start·stop·logs · drain 걸기·모드(graceful·at-boundary)
     S3b   건 뒤 — 현재 모드 배지 · 「풀기」(C3) · 「지금 도는 작업 없음」
     S5    멈춘 노드 — start 로 띄운다
     stop·재시작은 누르기 전에 그 Run 을 어떻게 끝내는지 말한다(cancel 먼저)
   transcript (CP6)
     S3    트랜스크립트 카드(도는 것) · 데몬 로그 카드 나란히(다른 물건)
           지난 작업 목록(이 노드) · 누르면 결과·봉인 트랜스크립트
   mcp (CP5·CP7 도구) — 화면 없음. 사용자 Claude 가 화면이다
```

---

## 4. 완결 정본

```text
   CP10   대회 데모 장면 완주 (가치 정본).  ui(데모 모드) · demo-back · 하드웨어
          아무나 게스트로 들어와 고정 시나리오 둘을 내고, rpi·mac 에서 돌며,
          웹캠으로 보드가 보이고, ① LED 새 패턴 · ② 「환영합니다」 재생·봉인.
          현황판이 보드의 사실과 맞다
```

**CP10 을 장면 완주 지점에 둔다** — 맨 끝에 두면 실패가 맨 끝에 온다. 기계 일곱
(CP0~CP7)이 그 아래 서고, 장면 밖 CP5·CP6·CP7 은 자리 지정된 뒤 아무 때나 돈다.
