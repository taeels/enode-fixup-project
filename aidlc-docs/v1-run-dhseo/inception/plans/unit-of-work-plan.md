# 유닛 분해 계획 — Units Generation (Part 1)

AI-DLC Units Generation 계획이다. **정본은 `requirements/` 팩**이고, 이 단계는
응용 설계(`aidlc-docs/inception/application-design/`) 위에서 **유닛의 수·경계·책임**과
**유닛 의존 그래프**, 그리고 팩이 필수로 요구한 **파일 행렬**을 낸다. 값은 팩과
응용 설계를 가리키고 다시 적지 않는다.

선행 맥락 — `requirements/` 팩 다섯 · `aidlc-docs/inception/reverse-engineering/`
아홉 · `application-design/` 다섯 · `plans/execution-plan.md` · `design/README.md` ·
데모 시안(`v2-run-shin_pen_drawing` 의 `design/exports/D1~D5`, 정합 검토 완료).

---

## 1. 이 단계가 정하는 것 · 안 정하는 것

**정한다.**

```text
   유닛의 수·경계·책임      새 기능을 어느 유닛이 지는가
   유닛 의존 그래프         병합 순서와 병렬 착수가 여기 걸린다 (CONVENTIONS 3.1)
   파일 행렬 (필수)         유닛마다 만지는 파일 · 둘 이상이 만지는 접점과 그 처리
                           (constraints 구조 불변식 — 팩이 산출물로 요구)
   유닛-게이트-기능 매핑     각 유닛이 어느 CP 를 재고 어느 기능을 지는가 (스토리 맵)
```

**안 정한다** — 건드리면 옮겨 적기가 된다.

```text
   메서드 안 비즈니스 로직   유닛별 Functional Design 몫 (정책 파일 형식 ·
                           at-boundary tx · 링 파일 로직 · QUEUED 재매칭 세부)
   팩이 값으로 닫은 것       폴링 · 바인딩 · 응답 모양 · 커버리지 80% · 차단 게이트 다섯
   열린 미정 둘             Capabilities{Caps,At} 읽기 계약(진행자 decisions) ·
                           sandbox 표시 출처(FD).  해당 유닛 FD 전에 닫힌다
```

---

## 2. 산출물 계획 (체크박스) — Part 2 가 생성

규칙(`units-generation.md` Step 2)의 필수 셋과 검증이다. 4절 답이 들어온 뒤 생성한다.

- [x] `application-design/unit-of-work.md` — 유닛 정의와 책임
  - [x] 유닛마다 목적 · 책임 · 주요 파일 · 재는 게이트 · 지는 기능
  - [x] 새 패키지 넷과 기존 패키지 확장이 어느 유닛에 드는지
- [x] `application-design/unit-of-work-dependency.md` — 의존 행렬 + **파일 행렬**
  - [x] 유닛 의존 행렬(누가 누구를 딛나) + 병렬 웨이브
  - [x] **파일 행렬**(유닛 x 파일) — 접점(둘 이상이 만지는 파일)과 그 처리
  - [x] 임포트 금지 넷이 어느 유닛에서 경계 검사 테스트로 서는지
- [x] `application-design/unit-of-work-story-map.md` — 유닛↔게이트↔기능
  - [x] CP0~CP11 을 유닛에 배정(단일 소유자 · 공동 게이트 · 회귀)
  - [x] 기능 일곱 + 시연 다섯이 전부 어느 유닛에 드는지 (누락 0)
- [x] 유닛 경계·의존 검증 — 임포트 금지 넷 · 접점 직렬 병합 규칙 · 모든 기능/게이트 배정 · 표기 규약

---

## 3. 제안 분해 (검토용) — 아홉 유닛

값은 응용 설계에서 왔고, 열린 경계 선택은 4절로 뺐다. **4절 답이 경계 셋을
움직일 수 있다** — 되묻기 접기(Q1) · 데모/UI 분할(Q2) · store 스키마 배치(Q3).

### 3.1 유닛 표

```text
   #  유닛              책임(요약)                                  게이트     의존
   ─  ───────────────   ─────────────────────────────────────────   ───────   ──────────────
   1  obs 관측 API       store Nodes·Runs·DrainingNodes·chosen·       CP1       (없음 · 토대)
                        requires·as·submitter 컬럼 · api nodes.go·
                        runs.go · getRun 확장 · runctl Client.Nodes·Runs
   2  queue 대기열       store QUEUED·CreateQueuedRun·WakeQueued·      CP2       obs
                        WakeQueuedNow · api submit 202 분기 +
                        DrainingNodes busy 합침(if !dry) · mediator 기동 wake
   3  drain             enode 정책 파일 읽기·광고 경로 · postResult    CP3       queue
                        at-boundary Cancel(drain:<node_id>) + WakeQueued
   4  board-ui 현황판    internal/api/ui S0~S2 정적 · cmd/mediator      CP1·CP2   obs
                        /ui/ embed 마운트 · 되묻기 카드 표시(여섯째 상태)  화면
   5  panel 제어판       internal/proc 추출 · internal/panel 서버 ·     CP4       obs · drain(정책
                        cmd/enodectl serve · cmd/enode panel 하위명령 ·          파일 형식)
                        경계 검사 테스트 · drain 토글 · 현재 작업
   6  transcript        enode 링 파일 tee(runner.go·claim.go) ·        CP6       panel · obs
                        panel 트랜스크립트 카드 · 지난 작업(runs 필터+record tar)
   7  mcp               internal/mcp · runctl mcp · 도구 열            CP5       obs
                        (asks.list·run.answer 포함) · runctl.Client 재사용    (+CP7 도구)
   8  demo-front        데모 대시보드 프론트 — 게스트 로그인 · 3D(S6·S7) ·  CP8·CP9   obs · board-ui ·
                        모달 · 웹캠 floating · 투어 · submitter 카드 표시   CP11 화면  demo-back
   9  demo-back         internal/api/demo.go — allow-list · 토큰       CP9       obs · queue
                        서버측 주입 · submitter 쓰기(Guest 로그인 이름)  서버측
```

되묻기(3.2.3)는 새 패키지·새 백엔드가 0이라 **독립 유닛으로 두지 않고** 표시는
board-ui, 도구 둘은 mcp 에 접었다 (Q1 권장 A). 온보딩 카드 시퀀스(3.4.1 첫 부분,
「왜 enode 인가」)는 **이 분해 밖**이다 — 별도 브랜치에서 작업 중이고 pen 에 있을
내용이 아니다(2026-09-08 사용자 결정). demo-front 의 CP8 지분은 게스트 로그인·투어·3D 다.

### 3.2 의존 그래프 · 병렬 웨이브

```text
   obs ─┬─> queue ─> drain ─> panel ─> transcript
        │                      ^
        ├─> board-ui ──────────┘(panel 이 board-ui 를 딛진 않음 · 화살표 정리는 아래)
        ├─> mcp
        └─> demo-back ─> demo-front
            board-ui ──────────> demo-front

   웨이브(병렬 착수)
     W0   obs                              토대.  모두가 딛는다 (임계 경로)
     W1   queue · board-ui · mcp           obs 만 딛는다 — 셋 병렬
     W2   drain(queue) · demo-back(obs·queue)
     W3   panel(obs·drain) · demo-front(obs·board-ui·demo-back)
     W4   transcript(panel·obs)

   가치 게이트 CP10 은 demo-front·demo-back 이 지고 W3 에 닫힌다 — 맨 끝이 아니다.
   CP7 은 board-ui(표시)·mcp(도구)가 W1 에 서므로 CP1 뒤 언제나 잰다.
```

panel 은 obs(runctl.Client.Nodes)와 drain(제어판이 쓰는 정책 파일 형식)을 딛는다.
drain 토글·CP4 완주를 위해 drain 이 필요하나, panel 의 나머지(proc·프로세스 제어·
현재 작업·신원)는 obs 만으로 착수할 수 있다 — **착수는 obs 뒤, 완료(CP4)는 drain 뒤**다.

### 3.3 파일 행렬 (접점 = 진행자 직렬 병합)

```text
   파일                          단독 소유 / 접점(유닛들)              처리
   ───────────────────────────   ─────────────────────────────────    ──────────────
   internal/store/store.go       접점: obs(읽기경로) · queue(QUEUED) ·  진행자 직렬
                                 demo-back(submitter 쓰기)
   internal/api/api.go           접점: obs(nodes·runs 등록+getRun) ·    진행자 직렬
                                 queue(submit 202) · demo-back(demo 등록)
   internal/api/ui (핸들러·자산)   접점: board-ui(S0~S2) · demo-front     진행자 직렬
                                 (데모 자산) — 자산 하위폴더는 갈리나 Handler 공유
   internal/panel (신규 패키지)    접점: panel(생성) · transcript(카드)    진행자 직렬
   cmd/mediator/main.go          접점: queue(기동 wake) · board-ui(/ui/ 마운트)  진행자 직렬
   internal/enode                조율: drain(정책·광고) · transcript(링 tee)  다른 파일 ·
                                                                        같은 패키지 커버리지
   internal/api/nodes.go·runs.go 단독: obs                              —
   internal/api/demo.go          단독: demo-back                        —
   internal/runctl/client.go     단독: obs (Nodes·Runs)                 —
   internal/proc/* (신규)         단독: panel                            —
   cmd/enodectl · cmd/enode      단독: panel                            —
   internal/mcp/* (신규) · cmd/runctl  단독: mcp                          —
   경계 검사 테스트                단독: panel (internal/panel 생성과 함께)  —
```

접점 다섯(store · api.go · api/ui · panel · mediator/main.go)은 **진행자가
직렬로 병합**한다 (CONVENTIONS 3.1·3.2). 유닛 브랜치는 자기 파일만 만지고
상태 파일 둘(`audit.md` · `aidlc-state.md`)과 `design/` 은 안 건드린다.

### 3.4 스토리 맵 요약 (유닛↔게이트↔기능)

```text
   CP0   회귀 — 모든 유닛이 착수·병합 전 재실행 (단일 소유자 없음)
   CP1   obs(API) + board-ui(화면)              3.1.1
   CP2   queue(API) + board-ui(화면 QUEUED·chosen) 3.2.1
   CP3   drain                                   3.2.2
   CP4   panel (obs·queue·drain·board-ui 협력)     3.1.2 (전환 후 정본은 CP10)
   CP5   mcp                                      3.3.1
   CP6   transcript                               3.1.3
   CP7   board-ui(표시) + mcp(도구)                3.2.3
   CP8   demo-front (게스트·투어·3D) [온보딩 카드=외부]  3.4.1·3.4.2
   CP9   demo-front(모달·전환·카드) + demo-back(allow-list·submitter)  3.4.3·3.4.4
   CP10  demo-front + demo-back + 하드웨어 (가치 정본)  전부
   CP11  demo-front (웹캠 floating)                3.4.5
```

기능 누락 점검 — 일곱(3.1.1·3.1.2·3.1.3·3.2.1·3.2.2·3.2.3·3.3.1)과 시연
다섯(3.4.1~3.4.5)이 전부 배정됐다. sandbox 표시(3.4.2)는 demo-front·board-ui 의
노드 표현에 자리를 두되 출처는 FD(§8.5).

### 3.5 답 반영 — 확정 분해 여덟 (Q1=A · Q2=C · Q3=A)

4절 답이 §3.1 제안(아홉)을 여덟으로 확정한다. 제안 표의 board-ui·demo-front 는
아래 ui 하나로 읽는다.

```text
   Q2=C  board-ui + demo-front 를 ui 한 유닛으로 합친다 — 실 함대 모드(S0~S2
         토큰·읽기전용)와 공개 데모 모드(게스트·3D·모달·웹캠·투어)를 한 표면의
         두 모드로.  internal/api/ui 를 단독 소유한다
   Q1=A  되묻기는 독립 유닛 안 둠 — 표시는 ui, 도구 둘(asks.list·run.answer)은 mcp
   Q3=A  store 스키마 기능별 분산 — 읽기경로(Nodes·Runs·chosen·requires·as·
         submitter 컬럼) obs · QUEUED 어휘·WakeQueued queue · submitter 쓰기
         demo-back.  store 는 접점(진행자 직렬 병합)

   확정 여덟    obs · queue · drain · ui · panel · transcript · mcp · demo-back

   ui           착수 obs(실 모드) · 완료 demo-back(데모 제출 라우트를 딛는다).
                게이트 CP1·CP2(실 모드) · CP7 표시 · CP8·CP9·CP11(데모 모드)
   접점 넷      store · api.go · panel · cmd/mediator/main.go (진행자 직렬).
                Q2=C 로 internal/api/ui 가 단독 소유가 되어 접점이 다섯에서 넷으로 준다
   웨이브       W0 obs · W1 queue·mcp·ui(실 모드) · W2 drain·demo-back ·
                W3 panel·ui(데모 모드) · W4 transcript
```

Part 2 가 이 확정 여덟로 산출물 셋(unit-of-work · unit-of-work-dependency[파일
행렬 포함] · unit-of-work-story-map)을 생성한다.

---

## 4. 결정이 필요한 것 — `[Answer]:` 태그

팩이 안 닫은 **유닛 경계** 셋이다. 각 질문에 권장(A)과 근거를 붙였다. `[Answer]:`
뒤에 고른 기호를 적어 주세요. 권장이 아니면 다른 기호나 직접 서술을 적으면 됩니다.

---

### Q1. 되묻기(3.2.3)를 독립 유닛으로 두나

되묻기는 새 패키지·새 Mediator 라우트·DB 변경이 **0**이다(3.2.3 데이터관리
「없다」). 실체가 (a) 현황판 카드의 여섯째 상태 표시 (b) MCP 도구 둘로 나뉜다.

```text
   A (권장)  접는다.  표시는 board-ui 유닛에, 도구 둘(asks.list·run.answer)은
             mcp 유닛에 넣는다.  CP7 은 두 유닛의 공동 게이트로 두고 각 완료
             조건에 나눠 박는다.  독립 유닛의 코드 실체가 얇아 값이 낮다
   B         독립 유닛으로 둔다.  CP7 을 단일 소유자가 진다.  대신 만지는 코드
             (현황판 자산 · MCP 도구 표)가 board-ui·mcp 와 겹쳐 접점이 둘 는다
```

근거 — A. 되묻기는 감싸는 표면이라 실체가 두 유닛에 자연히 걸린다. 접으면
접점이 안 늘고, CP7 은 화면(board-ui) · 도구(mcp)로 갈라 재면 「재는 기능」이 흐려지지 않는다.

**[Answer]:** A

---

### Q2. 데모 표면과 현황판 UI 를 몇 유닛으로 가르나

데모는 (a) 서버측 `internal/api/demo.go`(allow-list·토큰·submitter — Go, 커버리지
게이트) (b) 데모 프론트(게스트·모달·3D·웹캠 — `internal/api/ui` 자산, shonsin
pen·하드웨어)로 나뉘고, 실 현황판 UI(S0~S2)도 같은 `internal/api/ui` 에 산다.

```text
   A (권장)  셋으로 — board-ui(S0~S2 · CP1/CP2 화면) · demo-front(게스트·모달·
             3D·웹캠 · CP8/9/11 화면) · demo-back(demo.go 서버측 · CP9).
             백엔드는 Go 테스트로 커버리지를 지고 프론트는 pen·하드웨어에 걸려
             갈라야 병렬·게이트가 깨끗하다.  api/ui 는 board-ui·demo-front 접점
   B         둘로 — 데모(프론트+백엔드=demo) 하나 · 실 현황판 UI(board-ui) 하나
   C         둘로 — UI 전부(실 현황판+데모 프론트=board-ui) 하나 · 데모 백엔드
             (demo-back)만 서버측.  화면 자산을 한 유닛이 몰아 진다
```

근거 — A. demo-back 은 커버리지 게이트가 걸리는 서버 로직이고 demo-front 는
하드웨어·pen 에 걸린 화면이라, 갈라야 W2(백엔드)·W3(프론트) 병렬이 산다. 실
현황판(CP1)과 데모 대시보드(CP8~11)는 게이트도 화면도 달라 board-ui 와 분리가 자연스럽다.

**[Answer]:** C.

---

### Q3. store 스키마 변경(additive)을 어떻게 배치하나

store 에 QUEUED 어휘 · submitter 컬럼 · StepView.chosen · getRun requires·as
재료 · Nodes·Runs·DrainingNodes 가 는다. 3.1 제안은 이를 기능별로 흩었다
(obs·queue·demo-back).

```text
   A (권장)  기능별로 진다.  읽기 경로(Nodes·Runs·chosen·requires·as·submitter
             컬럼)는 obs, QUEUED 어휘·CreateQueuedRun·WakeQueued 는 queue,
             submitter 쓰기는 demo-back.  store 를 접점으로 표시하고 진행자가
             직렬 병합(CONVENTIONS 3.1 그대로).  스키마가 그 기능과 함께 착지한다
   B         스키마 우선 유닛(schema)을 맨 앞에 둬서 store 의 모든 additive
             DDL·컬럼·상수(QUEUED·submitter·chosen·requires/as 재료)를 한 번에
             낸다.  downstream 은 읽기/쓰기 로직만.  store 병합 충돌을 한 곳에
             모으고 마이그레이션을 한 번에 검토한다.  대신 obs·queue 가 그 유닛
             뒤에 착수(직렬 한 겹 추가) — W0 이 두 겹이 된다
```

근거 — A. additive 변경이라(새 테이블 0 · 파괴 변경 0) 기능별로 흩어도 충돌이
작고, 접점 직렬 병합 규칙이 이미 store 를 덮는다. 스키마 우선 유닛(B)은 마이그레이션
검토를 모으는 값이 있으나 임계 경로에 직렬 한 겹을 더한다 — 이틀 예산에서 비싸다.

**[Answer]:** A

---

## 5. 참고 — 이미 닫혀서 안 묻는 것 (확인용)

```text
   병합 순서·병렬       의존 그래프(3.2)가 정하고 진행자가 직렬 접점을 병합    CONVENTIONS 3.1
   접점 처리            store·api.go·api/ui·panel·mediator/main.go = 진행자 직렬  CONVENTIONS 3.2
   상태 파일·design     유닛이 안 건드린다.  진행자·shonsin 영역              CONVENTIONS 3.2
   온보딩 카드 시퀀스    이 분해 밖 · 별도 브랜치 · pen 내용 아님              사용자 결정 2026-09-08
   제출자 이름          Guest 로그인 이름 자동(모달 이름 칸 없음)             decisions §8.6
   웹캠                 out-of-band 임베드.  demo-front 안 · Mediator 밖      decisions §8.2 Q4
   임포트 금지 넷        경계 검사 테스트는 panel 유닛이 낸다(CP0 아님)         constraints
   3D 자산 S6·S7        shonsin pen(D2·D5).  UI 유닛이 소비만               design
```

---

## 6. 확장 준수 (security-baseline · Enabled)

| 확장 | 상태 | 이 단계 판정 |
|---|---|---|
| security-baseline | Enabled | 준수. 이 단계는 분해 문서만 내고 코드를 안 만든다. 데모 공개 쓰기(SECURITY-08 수락 위험)를 **demo-back 한 유닛**에 가둬 allow-list·서버측 토큰 주입·일회용 Mediator 의 가둠 셋이 한 경계 안에 서게 했다. 제어판 LAN 토큰(SECURITY-12)은 panel 유닛이 진다. 산출 코드가 없어 객체 수준 권한 등 세부는 N/A. 차단 소견 없음 |
| resiliency-baseline | Disabled | 건너뜀 (`decisions §1`). audit 스킵 기록 |
| property-based-testing | Disabled | 건너뜀 (`decisions §1`). audit 스킵 기록 |

---

## 7. 다음

`[Answer]:` 셋이 채워지면 — (1) 답을 분석해 모호·모순·결합을 본다, (2) 있으면
후속 질문을 이 문서에 더한다, (3) 없으면 2절 산출물 셋(`unit-of-work.md` ·
`unit-of-work-dependency.md`(파일 행렬 포함) · `unit-of-work-story-map.md`)을
생성하고 완료 게이트를 연다.
