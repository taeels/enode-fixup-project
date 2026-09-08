# 응용 설계 계획 — Application Design

AI-DLC Application Design 산출물 계획이다. **정본은 `requirements/` 팩**이고,
이 단계는 팩이 **일부러 안 준 것**을 짓는다 — 새 표면의 메서드 겉면과 컴포넌트
경계다. 팩은 새 표면의 모양과 임포트 금지만 제약으로 주고 메서드 설계를 이
단계로 넘겼다 (`execution-plan.md` 165~170 · `constraints.md` 구조 불변식).

선행 맥락 — `requirements/` 팩 다섯 · `aidlc-docs/inception/reverse-engineering/`
아홉 · `design/README.md`.

---

## 1. 이 단계가 정하는 것 · 안 정하는 것

**정한다** (실행 계획 165~170).

```text
   새 패키지 셋의 겉면        internal/panel · internal/api/ui · internal/mcp
                             Config · Server · New 와 메서드 시그니처
   제어판의 Mediator 접근     제어판이 관측 API 를 어느 클라이언트로 부르나
   MCP 도구 열의 디스패치     열 개 도구가 기존 REST 로 어떻게 내려가나
   store.WakeQueued          시그니처와 여섯 호출 지점의 계약
   QUEUED 전이               submit 두 자리가 QUEUED 행을 어떻게 남기나
   데모 표면의 경계           정적 자산 vs 서버측 핸들러가 갈리는 자리
```

**안 정한다** — 두 부류다. 건드리면 옮겨 적기가 된다.

```text
   팩이 값으로 닫은 것        폴링 5초 · 바인딩 127.0.0.1:8081 · 응답 모양 ·
                             필터 넷 · 링 파일 설계 · 커버리지 80% ·
                             차단 게이트 다섯.  decisions.md 가 진다
   진행자·FD 로 이월한 계약    Capabilities{Caps,At} 읽기 계약(진행자가
                             decisions 에 행을 더한다) · sandbox 표시 출처(FD).
                             아래 6절에 열린 미정으로 적고 이 단계는 자리만 남긴다
```

이 단계는 **메서드 수준의 비즈니스 로직을 안 짠다** — 그것은 유닛별 Functional
Design 이다. 여기서는 겉면(시그니처·책임·의존)까지다.

---

## 2. 산출물 계획 (체크박스)

규칙이 요구하는 필수 산출물 다섯과 검증이다. 4절 질문의 답이 들어온 뒤 생성한다.

- [x] `application-design/components.md` — 컴포넌트 정의와 책임
  - [x] 새 패키지 넷(panel · api/ui · mcp · proc) · 새 핸들러 파일 셋
  - [x] 만지는 기존 패키지 여섯의 확장 책임
  - [x] 각 컴포넌트의 인터페이스 요약
- [x] `application-design/component-methods.md` — 메서드 시그니처 (비즈 규칙은 FD)
  - [x] `internal/panel` — Config · Server · New · 핸들러 겉면
  - [x] `internal/mcp` — Server · 도구 디스패치 표 · 열 개 도구 겉면
  - [x] `internal/api/ui` — embed FS · Handler · (데모 자산 자리)
  - [x] `internal/store` — `WakeQueued` · QUEUED 생성/전이 겉면
  - [x] `internal/runctl.Client` 확장 (Q2=A) · `internal/api` 새 핸들러 겉면
- [x] `application-design/services.md` — 서비스 정의와 오케스트레이션
  - [x] 시퀀싱 — `WakeQueued` 여섯 지점 · QUEUED 전이 흐름
  - [x] drain 경로 (광고 왕복 · at-boundary 취소) 오케스트레이션
  - [x] MCP stdio 디스패치 루프 · 데모 제출 오케스트레이션
- [x] `application-design/component-dependency.md` — 의존 행렬과 통신 패턴
  - [x] 의존 행렬 (임포트 금지 넷을 표에 박는다)
  - [x] 통신 패턴 (HTTP · stdio · 파일 tee · 프로세스 신호)
  - [x] 데이터 흐름도
- [x] `application-design/application-design.md` — 위 넷의 통합본
- [x] 설계 완결성·정합 검증 (임포트 금지 · 차단 게이트 다섯 · 겉면 얇음 확인 — 통과)

---

## 3. 제안 설계 요약 (검토용)

4절 질문에 답하기 전에 전체 모양을 본다. 값은 팩에서 왔고, 열린 선택은 4절로
뺐다.

### 3.1 컴포넌트 — 새것 · 만지는 것

```text
   새 패키지
     internal/panel     호스트 제어판 HTTP 서버.  Mediator 를 HTTP 로 본다.
                        store · api 를 임포트 안 한다
     internal/api/ui    /ui/ 아래 정적 화면(현황판 · 데모 대시보드 · 3D · 온보딩).
                        embed FS 하나.  store 를 임포트 안 한다
     internal/mcp       runctl mcp — stdio JSON-RPC.  기존 REST 를 감싼다

   새 파일 (기존 패키지 안)
     internal/api/*.go  새 핸들러 — GET /v1/nodes · GET /v1/runs 목록 ·
                        데모 제출.  api.go 에는 등록 줄만 (constraints)

   만지는 기존 패키지
     internal/store     QUEUED 어휘 · WakeQueued · submitter · draining 제외 재료
     internal/match     busy 에 DrainingNodes 합침 (if !dry 안)
     internal/enode     drain 정책 파일 읽기 · 광고 경로 · 링 파일 tee · 링 읽기
     cmd/enodectl       serve 하위명령 (위임 exec — 심볼 상한 회피)
     cmd/mediator       /ui/ embed 마운트 · 데모 라우트 등록
     internal/runctl    (Q2) 관측 라우트 클라이언트 메서드
     cmd/enode          panel 하위명령 (enodectl serve 가 exec 하는 대상)
```

### 3.2 임포트 불변식 (constraints — 검사기가 표를 읽는다)

```text
   internal/panel   -> internal/store    금지
   internal/panel   -> internal/api      금지
   internal/api/ui  -> internal/store    금지
   internal/enode   -> internal/panel    금지 (반대 방향 cmd/enode -> panel 은 허용)
```

경계 검사 테스트는 `internal/panel` 을 만드는 유닛이 함께 낸다 (CP0 아님 —
그때는 검사할 패키지가 없다).

### 3.3 새 표면의 겉면 (Config · Server · New — 얇게)

제안이다. 시그니처의 인자·반환은 4절 답으로 굳는다.

```text
   internal/panel
     type Config struct { Listen string; Node string; MediatorBase, Token string; ... }
     type Server struct { ... }
     func New(cfg Config) (*Server, error)
     func (s *Server) Handler() http.Handler       // /api/* · 정적
     // 현재 작업 · drain 토글 · 트랜스크립트 · 프로세스 제어 핸들러

   internal/mcp
     type Server struct { ... }                      // internal/runctl.Client 를 쥔다
     func New(client *runctl.Client) *Server
     func (s *Server) Serve(ctx, in io.Reader, out io.Writer) error   // stdio 루프
     // 도구 열 개: dispatch 표 (도구 이름 -> 핸들러)

   internal/api/ui
     //go:embed assets
     func Handler() http.Handler                     // 정적.  store 를 안 부른다
```

### 3.4 시퀀싱 (services 미리보기)

```text
   QUEUED 전이       submit 두 자리(매처 거절 api.go:398~408 · ErrNodeTaken
                     롤백 api.go:457~463)가 ALLOCATING -> QUEUED.  202 응답
   WakeQueued        임대가 지워지는 여섯 지점 뒤에서 부른다 (decisions §1):
                     postResult · postCancel · Reap · FailRestarted->SettleIfDone ·
                     applyStepEffects 부분 반납 · drain 해제 광고 처리.
                     부르는 요청의 트랜잭션 안에서 동기 (CP2)
   drain 경로        정책 파일 -> 광고 -> UpsertAdvert nodes 열 -> 광고 응답 ->
                     Worker.  at-boundary 는 postResult 끝에서 store.Cancel
   MCP 디스패치       stdio JSON-RPC 루프 -> 도구 이름 -> Client 메서드 -> REST
   데모 제출         /ui/ 옆 서버측 라우트 -> allow-list 검사 -> 토큰 주입 ->
                     내부 submit.  브라우저에 실 토큰 없음 (SECURITY-08 가둠)
```

---

## 4. 결정이 필요한 것 — `[Answer]:` 태그

팩이 안 닫은 **메서드·경계 설계**의 열린 선택 다섯이다. 각 질문에 권장(A)과
근거를 붙였다. `[Answer]:` 뒤에 고른 기호(또는 직접 서술)를 적어 주세요.
권장을 그대로 받으면 `A` 만 적어도 됩니다.

---

### Q1. 공용 프로세스 제어 원장을 어디에 두나

`enode-features 3.1.2` 는 플랫폼으로 갈린 프로세스 제어 짝
(`proc_unix.go`/`proc_windows.go` 의 `processAlive` · `signalStop` · `ownsConfig`)을
`internal/panel` 로 내려서 제어판과 CLI 가 **같이 쓴다**고 적었다. 그런데 차단
게이트가 이것과 부딪친다 — `cmd/enodectl` 이 `internal/panel` 을 임포트하면
`enodectl.exe` 의 링크 그래프가 `internal/panel` 의 `net/http` 에 닿아 심볼
상한(net/http ≤50)이 빨개진다. 「같이 쓴다」와 「심볼 상한」이 한 패키지에서
같이 성립하지 않는다.

```text
   A (권장)  새 공용 패키지 internal/proc (이름 조정 가능).  프로세스 원장과
             빌드 태그 짝을 여기 두고 cmd/enodectl 과 internal/panel 이 둘 다
             임포트한다.  proc 는 net/http 를 안 쓰므로 enodectl.exe 는 깨끗하다.
             빌드 태그 짝이 한 벌로 유지되고 「같이 쓴다」가 지켜진다
   B         원장을 cmd/enodectl 에 그대로 두고 internal/panel 이 자기 짝을
             따로 둔다 (중복).  공용 패키지가 안 늘지만 빌드 태그 짝이 두 벌이 된다
   C         원장을 internal/enode 에 둔다.  단 enode 는 데몬이라 결이 다르고
             cmd/enodectl 이 internal/enode 를 임포트하는 비용을 확인해야 한다
```

근거 — A 는 팩의 「같이 쓴다」를 지키면서 심볼 상한을 피하는 유일한 길이다
(`cmd/enodectl` 은 `package main` 이라 임포트 대상이 못 되므로 공유하려면
`internal/*` 로 내려야 하고, 그 패키지는 `net/http` 를 안 물어야 한다).
심볼 상한을 코드로 어떻게 넘기는지의 세부는 FD 몫이지만, 원장이 사는 패키지
경계는 이 단계가 정한다.

**[Answer]:** A

---

### Q2. 제어판과 MCP 가 Mediator 를 부르는 클라이언트

`internal/panel` 의 「현재 작업」은 Mediator 조회로 채우고(`decisions §2`),
`internal/mcp` 는 열 개 도구가 전부 기존 REST 를 감싼다. 둘 다 HTTP 클라이언트가
필요하다. `internal/runctl/client.go` 가 이미 `Submit` · `Status` · `Cancel` ·
`Record` · `Capabilities` · `Asks` · `Answer` 를 갖고 있다 (`decisions §5.2` 가
MCP 의 재사용 대상으로 지목). 다만 `GET /v1/nodes` · `GET /v1/runs` 목록은 이
회차가 만드는 새 라우트라 클라이언트에 메서드가 없다.

```text
   A (권장)  internal/runctl.Client 를 재사용하고 Nodes() · Runs() 두 메서드를
             거기 더한다.  panel 과 mcp 가 같은 클라이언트를 쥔다.
             임포트 금지에 안 걸린다 (panel/mcp -> runctl 은 store/api 가 아니다)
   B         panel · mcp 가 각자 얇은 HTTP 클라이언트를 따로 둔다.
             격리는 늘지만 do()/인증/에러 처리가 세 벌이 된다
   C         A 로 하되 Nodes()/Runs() 는 원문 JSON 을 그대로 돌려주는 형태로
             더한다 (Q3 과 묶임)
```

근거 — A 가 `decisions §5.2`(「HTTP 클라이언트를 새로 안 짠다」)와 「새 표면은
얇게」에 맞는다. `internal/runctl` 은 금지 표의 store·api 가 아니므로 panel·mcp
가 임포트해도 불변식이 안 깨진다.

**[Answer]:** A

---

### Q3. MCP `fleet.list` · `runs.list` 의 글자 일치 방법

CP5 가 「`fleet.list` 가 `GET /v1/nodes` 와 **글자까지 같다**」를 잰다. 도구가
낸 자료가 현황판이 그리는 자료와 한 글자도 다르면 안 된다.

```text
   A (권장)  두 목록 도구는 Mediator 응답 본문(JSON)을 그대로 흘려보낸다
             (raw passthrough).  디코드·재직렬화를 안 하므로 키 순서·공백까지
             동일이 보장된다
   B         응답을 타입 DTO 로 디코드했다가 다시 직렬화한다.  깔끔하지만
             키 순서·수치 표현이 미세하게 달라져 「글자까지」가 흔들릴 위험
   C         A 를 두 목록 도구에만 적용하고, 나머지 여덟 도구는 기존 Client
             메서드의 타입 반환을 쓴다 (혼합)
```

근거 — A(또는 목록만 A 인 C)가 CP5 의 「글자까지 같다」를 설계로 보장한다.
재직렬화는 같은 값을 다르게 쓸 수 있어 게이트를 취약하게 만든다.

**[Answer]:** A

---

### Q4. `store.WakeQueued` 의 계약 (범위·반환)

`decisions §1` 이 행동을 닫았다 — 임대가 지워지는 여섯 지점 뒤에서 부르고,
**부르는 요청의 트랜잭션 안에서 동기로** 돈다 (CP2 가 「a 가 끝난 뒤
`runctl status <b>` -> RUNNING」을 요구하므로 비동기면 경쟁이 된다). 남은 것은
메서드 겉면이다 — 무엇을 훑고 무엇을 돌려주나.

```text
   A (권장)  func (s *Store) WakeQueued(ctx, tx) ([]string, error)
             QUEUED 를 FIFO(created_at 오름차순)로 훑어 지금 매칭되는 것을
             RUNNING 으로 올리고, 올린 run_id 들을 돌려준다.  호출자 트랜잭션(tx)을
             받아 같은 트랜잭션에서 돈다.  FIFO 공정성(constraints §2)과 동기
             보장(CP2)을 둘 다 만족한다
   B         노드 한 대가 풀렸으니 그 노드에 맞는 첫 QUEUED 하나만 올린다.
             가볍지만 「여러 노드가 한 번에 풀리는」 지점(Reap · drain 해제)에서
             남은 대기 Run 이 다음 종료까지 멎을 수 있다
   C         반환 없이 부작용만 (올린 수도 안 돌려줌).  호출자가 결과를 못 보고
             테스트가 「무엇이 깨어났나」를 관측 못 한다
```

근거 — A 가 여섯 지점 전부에서 옳게 돈다. 특히 만료 회수(`Reap`)와 drain
해제는 여러 임대를 한꺼번에 지울 수 있어 FIFO 전체 훑기가 필요하다. 반환값은
동기 보장을 테스트로 재는 데 쓴다 (CP2).

**[Answer]:** A

---

### Q5. 데모 제출 엔드포인트와 allow-list 가 사는 자리

`decisions §8.2 Q2` — 데모 제출은 서버측 allow-list 라우트가 받아 고정
시나리오만 통과시키고 **토큰을 서버측에서 주입**한다 (브라우저에 실 토큰 없음 —
SECURITY-08 가둠 셋 중 하나). 이건 정적 파일이 아니라 서버 로직이다 —
`internal/api/ui` 는 정적이고 `store` 임포트가 금지라 여기 못 산다.

```text
   A (권장)  internal/api 에 새 핸들러 파일 하나 (예: demo.go).  api.go 에는
             등록 줄만 (constraints 「api.go 는 등록 줄만」).  고정 시나리오
             픽스처는 internal/contract 의 example 에서 온다 (runctl example ·
             decisions §8.3).  allow-list 는 그 픽스처 이름 집합
   B         새 패키지 internal/demo 가 allow-list·주입을 지고 internal/api 가
             얇게 위임한다.  「새 기능은 새 패키지」에 더 맞지만, 제출은
             store·match 를 타므로 api 와 결국 같은 자리를 만진다
   C         A 로 하되 데모 정적 자산(대시보드 개조·온보딩·3D·웹캠 임베드)은
             internal/api/ui 안에 함께 둔다 (서버측은 A, 프론트는 ui)
```

근거 — 데모 제출은 `store`·`match` 를 타는 서버 로직이라 `internal/api` 가
자연스러운 자리이고, 새 핸들러는 새 파일 · `api.go` 는 등록 줄만이라는 팩 규칙에
그대로 맞는다 (A). 정적 자산은 어차피 `internal/api/ui` 이므로 실무 형태는
A+정적분리 = C 에 가깝다. B 를 고르면 경계가 하나 더 는다.

**[Answer]:** A

---

## 5. 참고 — 이미 닫혀서 안 묻는 것 (확인용)

아래는 선례·팩이 정해서 **질문이 아니라 설계에 그대로 박는다**. 다르게 가려면
근거를 적는다 (`decisions.md` 게이트 규칙).

```text
   토큰 제시        Authorization: Bearer          internal/runctl/client.go:64
   에러 본문        {"error":{"code","reason"}}     internal/api 의 fail
   만료 필터        expires_at > now()             store.go:112
   플랫폼 분기      !windows / windows 빌드 태그 짝
   상태 술어        Terminal = SUCCEEDED · FAILED
   제어판 바인딩     127.0.0.1:8081 · LAN 은 --listen  decisions §2
   폴링 간격        Mediator 5초 · 트랜스크립트 1초    decisions §2 · §6.2
   링 파일 설계      512 KiB WriteAt · 잠금·rename·삭제 없음  decisions §6.3
   현황판 embed      /ui/ 정적 · cmd/mediator 마운트    decisions §2
   MCP 전송·토큰     stdio 하나 · 환경변수             decisions §5.2
   getRun 확장      requires · as · StepView.chosen   ADR-069 · ADR-060
```

---

## 6. 열린 미정 — 이 단계가 강제하지 않는다

`execution-plan.md` 243~254 가 해당 유닛의 Functional Design 전에 닫는다고 적은
둘이다. Application Design 은 자리(seam)만 남기고 값을 안 박는다.

```text
   1  제어판이 데몬 Capabilities{Caps,At} 를 읽는 계약 (ADR-068 · canon §1).
      어느 파일·어느 라우트·어느 모양인지의 계약 물음.  진행자가 decisions.md
      에 행을 더해 닫는다.  이 단계는 internal/panel 에 「탐지 능력 · 잰 시각」을
      내는 메서드 자리만 두고 출처를 TBD 로 남긴다
   2  sandbox 표시 출처 (decisions §8.5).  광고에 싣는지 · 계약의 sandbox: 를
      읽는지 — 있는 값을 읽는다.  FD 가 정한다.  노드 표현 컴포넌트가 딛는다
```

두 미정이 이 단계의 승인 차단 요인은 아니다 — 겉면에 자리를 남기는 것으로
충분하고, 값은 해당 유닛 FD 전에 닫힌다.

---

## 7. 확장 준수 (security-baseline · Enabled)

`aidlc-docs/aidlc-state.md` 의 Extension Configuration 을 확인했다.

| 확장 | 상태 | 이 단계 판정 |
|---|---|---|
| security-baseline | Enabled | 준수. 이 단계는 설계 문서만 내고 코드를 안 만든다. 설계가 결정된 보안 태세를 안 무른다 — 새 표면 겉면에 SECURITY-08 가둠 셋(일회용 데모 Mediator · 서버측 allow-list · 브라우저에 실 토큰 없음 — Q5)과 SECURITY-12(제어판 LAN 토큰 · 5절)를 설계로 박는다. 객체 수준 권한 등 산출 코드에 걸리는 세부는 이 단계에 코드가 없어 N/A. 차단 소견 없음 |
| resiliency-baseline | Disabled | 건너뜀 (`decisions §1`). audit 에 스킵 기록 |
| property-based-testing | Disabled | 건너뜀 (`decisions §1`). audit 에 스킵 기록 |

---

## 다음

`[Answer]:` 다섯이 채워지면 — (1) 답을 분석해 모호·모순·결합을 본다, (2) 있으면
후속 질문을 이 문서에 더한다, (3) 없으면 2절 산출물 다섯 문서를 생성하고 완료
게이트를 연다.
