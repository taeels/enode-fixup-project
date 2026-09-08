# 응용 설계 — 통합본

AI-DLC Application Design 산출물의 통합본이다. 상세 넷을 한 자리에 묶는다.
**정본은 `requirements/` 팩**이고 이 설계는 팩이 이 단계로 넘긴 것 — 새 표면의
메서드 겉면과 컴포넌트 경계 — 을 짓는다.

```text
   components.md            컴포넌트 정의와 책임
   component-methods.md     메서드 시그니처 (비즈 규칙은 FD)
   services.md              시퀀싱과 오케스트레이션
   component-dependency.md  의존 행렬 · 통신 패턴 · 데이터 흐름
```

선행 — `requirements/` 팩 다섯 · `aidlc-docs/inception/reverse-engineering/` 아홉 ·
`design/README.md` · `application-design-plan.md`.

---

## 1. 무엇을 짓나 (한눈에)

관측·제어·접수 기능 일곱과 대회 데모 표면 다섯을 기존 Go 모노레포에 더한다.
아키텍처 전환이 아니라 기능 확장이다 (`packaging/` 무변경).

```text
   새 패키지 넷    internal/panel · internal/api/ui · internal/mcp · internal/proc
   새 핸들러 셋    internal/api/{nodes,runs,demo}.go  (api.go 는 등록 줄만)
   만지는 기존     store · match · enode · runctl · cmd/{enodectl,enode,mediator}
   새 상태 하나    QUEUED (runs.state 넷 -> 다섯)
   새 필드         runs.submitter · StepView.chosen · getRun 의 requires·as
   기존 라우트 15   전부 보존.  신규는 GET /v1/nodes · GET /v1/runs · 데모 제출
```

---

## 2. 결정 다섯 (application-design-plan.md 4절 · 전부 A)

```text
   Q1  공용 프로세스 원장     새 패키지 internal/proc.  proc_*.go 짝을 여기 내려
                            enodectl 과 panel 이 같이 쓴다.  net/http 를 안 물어
                            enodectl.exe 심볼 상한이 안 깨진다
   Q2  Mediator 클라이언트    internal/runctl.Client 재사용 + Nodes·Runs 메서드.
                            panel·mcp 가 같은 클라이언트를 쥔다
   Q3  MCP 글자 일치          fleet.list·runs.list 는 응답 원문을 그대로 흘려보낸다.
                            CP5 의 「글자까지 같다」를 설계로 보장
   Q4  WakeQueued 겉면        (ctx, tx) ([]string, error).  QUEUED 를 FIFO 로 훑어
                            승격하고 올린 run_id 를 돌려준다.  FIFO+동기 둘 다 만족
   Q5  데모 제출 자리         internal/api 새 핸들러(demo.go) + 픽스처는 contract
                            example.  정적 자산은 internal/api/ui.  api.go 는 등록 줄만
```

---

## 3. 컴포넌트 요약

새 패키지 넷:

```text
   internal/panel     호스트 제어판 HTTP 서버.  Config·Server·New·Handler.
                      Mediator 를 runctl.Client 로 HTTP 로만 본다.  proc 로 프로세스 제어
   internal/api/ui    /ui/ 정적 화면 (현황판 S0~S2 · 데모 대시보드 · 3D S6·S7 ·
                      온보딩 · 웹캠 임베드).  embed FS + Handler.  서버 로직 없음
   internal/mcp       runctl mcp — stdio JSON-RPC.  도구 열 개가 기존 REST 를 감싼다.
                      Server·New(client)·Serve.  새 Go 의존 0
   internal/proc      플랫폼 프로세스 원장 (processAlive·signalStop·ownsConfig).
                      net/http 안 씀.  enodectl·panel 이 공유
```

만지는 기존 여섯:

```text
   store     QUEUED · WakeQueued · CreateQueuedRun · Nodes · Runs · DrainingNodes ·
             submitter · StepView.chosen · getRun requires·as
   match     시그니처 무변경.  draining 제외는 api 가 busy 에 합침 (if !dry)
   enode     drain 정책 파일 읽기 · 광고 경로 · 하네스 링 파일 tee/읽기
   runctl    Client 에 Nodes·Runs 추가 (원문 반환)
   enodectl  serve <name> — enode 제어판을 exec 위임 (심볼 상한 회피)
   enode(cmd) 제어판 하위명령 — internal/panel 을 띄운다 (허용된 방향)
   mediator  /ui/ embed 마운트 · 새 라우트 등록
```

---

## 4. 임포트 불변식 (검사기가 표를 읽는다)

```text
   internal/panel   -> internal/store    금지
   internal/panel   -> internal/api      금지
   internal/api/ui  -> internal/store    금지
   internal/enode   -> internal/panel    금지 (cmd/enode -> panel 은 허용)
```

경계 검사 테스트는 `internal/panel` 을 만드는 유닛이 함께 낸다 (CP0 아님).
`panel/mcp -> runctl` 과 `enodectl/panel -> proc` 는 금지에 안 걸린다.

---

## 5. 시퀀싱 (services.md 요약)

```text
   관측       세 표면(현황판·제어판·MCP)이 GET /v1/nodes · GET /v1/runs 를 딛는다
   QUEUED     submit 두 자리(매처 거절 · ErrNodeTaken)가 CreateQueuedRun -> 202
   깨우기     임대 삭제 여섯 지점 뒤 WakeQueued — 부르는 tx 안에서 동기 (CP2)
   drain      정책 파일 -> 광고 -> nodes 열 -> 응답 -> Worker.  중앙 라우트 없음.
              at-boundary 는 postResult 끝에서 Cancel(drain:<node_id>)
   트랜스크립트  링 파일 1초 폴링 (도는 것) + GET /v1/runs·record tar (지난 것).  DB 안 만짐
   되묻기      현황판은 GET /v1/asks 를 보이기만.  답은 runctl·MCP.  중앙은 안 받음
   MCP        stdio 루프 -> 도구 -> Client -> REST.  글자 일치는 원문 passthrough
   데모        allow-list -> 토큰 서버측 주입 -> 내부 submit.  브라우저에 실 토큰 없음
```

---

## 6. 보안 태세 (security-baseline · Enabled)

설계에 박는 것 (`decisions §3·§8`):

```text
   제어판 바인딩     127.0.0.1:8081 기본.  LAN 은 --listen + panel_token 필수 (SECURITY-12)
   토큰 경계        Mediator 토큰 재사용 안 함.  MCP·데모는 토큰을 인자·브라우저에 안 냄
   데모 공개 쓰기     SECURITY-08 수락 위험 — 가둠 셋(일회용 Mediator · 서버측
                    allow-list · 브라우저에 실 토큰 없음)을 설계로 박는다
   관측 보호        GET /v1/nodes · GET /v1/runs 는 기존 Mediator 토큰 규칙.
                    /ui/ 정적은 무인증
   기존 코드 사실     SECURITY-01·03·07 은 기록만 (수정 안 함 · decisions §3)
```

이 단계는 설계 문서만 내고 코드를 안 만든다. 객체 수준 권한 등 산출 코드에
걸리는 세부는 N/A. 차단 소견 없음.

---

## 7. 열린 미정 (이 단계가 강제하지 않음)

```text
   1  제어판이 Capabilities{Caps,At} 를 읽는 계약 (ADR-068).  진행자가
      decisions.md 에 행을 더해 해당 유닛 FD 전에 닫는다.  panel 에 자리만 남김
   2  sandbox 표시 출처 (decisions §8.5).  있는 값을 읽는다.  FD 가 정한다
```

둘 다 승인 차단 요인이 아니다 — 겉면에 자리를 남기는 것으로 충분하다.

---

## 8. 다음 단계로 넘기는 것

```text
   Units Generation   파일 행렬(필수) + 유닛 의존 그래프.  유닛의 수·경계·병합
                      순서를 이 설계 위에서 정한다 (constraints)
   Functional Design   정책 파일 형식 · at-boundary 취소 tx · 링 파일 로직 ·
                      QUEUED 재매칭 세부 · allow-list 픽스처 · Capabilities/sandbox
   Code Generation     조각 게이트 명령을 계획에 박는다 (CP0~CP11)
```

이 설계는 **겉면까지**다. 메서드 안의 비즈니스 로직은 유닛별 Functional Design
이 짠다.
