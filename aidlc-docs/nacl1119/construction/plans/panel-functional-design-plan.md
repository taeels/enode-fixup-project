# panel — Functional Design 계획

AI-DLC Construction · **panel 유닛**(W3 · CP4 · 담당 nacl1119)의 Functional
Design Part 1 이다. 설계 본문이 아니라 **무엇을 설계할지의 계획과 물음**이다.
ADR-068 답(진행자 · A)이 왔으므로 그 값으로
`aidlc-docs/nacl1119/construction/panel/functional-design/` 아래 산출물을 낸다.

정본 — 유닛 정의 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md` §5 ·
게이트 `requirements/scene-gates.md` CP4(43행 · 90~99행) · 팩 `requirements/`
(enode-features 3.1.2 · decisions §2 「제어판 서버 위치」·「탐지 능력 읽기(ADR-068)」·
「제어판 LAN 노출의 토큰」·§6.5 · constraints) · 착수 길잡이
`aidlc-docs/nacl1119/construction/plans/panel-transcript-handoff.md`.

---

## 0. 이 유닛이 지는 것 (한눈)

```text
   신규 패키지   internal/proc     processAlive · signalStop · ownsConfig 를
                                  cmd/enodectl/proc_*.go 에서 내린다.  빌드 태그 짝 유지 · net/http 안 씀
                 internal/panel    제어판 서버.  127.0.0.1:8081 기본.  New(cfg) · Handler()

   신규 명령     cmd/enodectl serve <name>   enode 제어판 프로세스를 exec 위임(심볼 상한 회피)
                 cmd/enode 제어판 하위명령    internal/panel 을 net/http 로 띄운다

   신규 테스트   임포트 경계 검사 테스트 (internal/panel 을 처음 만드는 유닛이 낸다)

   딛는 표면     runctl.Client.Nodes(원문) · Cancel · Capabilities  (obs · main 병합됨)
                 internal/enode/policy.go 의 정책 파일 형식 (drain · shin-son 병합됨)
                 데몬이 쓰는 로컬 상태 파일 <stem>.status.yaml (ADR-068=A · 새로 냄)

   지는 기능  enode-features 3.1.2.  재는 게이트 CP4 (2일차 정오 · 가장 앞에 당긴 게이트)
   의존       obs(runctl.Client.Nodes · 착수) · drain(제어판이 쓰는 정책 파일 형식 · 완료).  둘 다 닫힘
```

---

## 1. 착수 전에 실측한 것 (기존 코드)

계획을 세우기 전에 저장소를 읽었다.

```text
   확인한 사실                                                        자리
   ─────────────────────────────────────────────────────────────     ────────────────────────────────
   internal/panel · internal/proc 둘 다 아직 없다                       ls (신규)
   proc 함수 넷이 cmd/enodectl 안에 빌드 태그 짝으로 산다               cmd/enodectl/proc_unix.go · proc_windows.go
     processAlive · signalStop · ownsConfig 를 내린다.  그런데 같은
     파일이 detachAttr · signalKill · exeSuffix · (윈도우)상수들도 쥐고
     있고 그것은 cmdStart 가 쓴다 — 추출 경계를 FD 에서 못 박는다
   cmdStart 가 detachAttr, cmdStop 이 signalStop·signalKill,             cmd/enodectl/main.go:261,293,304,183
     ownsConfig 가 pid 검사에 쓰인다 — 추출 뒤에도 이 호출들이 산다
   cmdSetup 이 이미 exec 위임의 본보기다 — enodeBin() 을 찾아             cmd/enodectl/setup.go:28
     exec.Command 로 넘기고 종료 코드를 그대로 되쓴다.  serve 가 이것을 본뜬다
   데몬(enode)은 나가는 http.Client 만 있고 자기 서버가 없다             cmd/enode/main.go:155
   데몬은 하위명령 분기를 os.Args[1] 로 한다 (hook · setup · --version).  cmd/enode/main.go:44~59
     제어판 하위명령이 여기 붙는다
   Capabilities{Caps, At} 는 데몬 메모리에만 있다 (Detector)             internal/enode/detector.go:31~86
   advertise.go 가 광고 직전마다 정책 파일을 읽는다(readPolicy).          internal/enode/advertise.go:159,167
     같은 자리에서 snap.At 이 손에 있다 — status 파일 쓰기의 자연스러운 자리
   Policy 구조체가 키 하나(drain)이고 policyReader 가 모르는 키를         internal/enode/policy.go:24,58
     무시한다 — panel_token 을 더해도 데몬을 안 건드린다
   PolicyPath(configPath) 가 <dir>/<stem>.policy.yaml 규칙을 진다        internal/enode/policy.go:32
   runctl.Client.Nodes 는 원문 json.RawMessage 를 낸다                   internal/runctl/client.go:337
     — panel 은 자기 node_id 행의 lease 를 뽑는다 (parse 는 panel 몫)
   runctl.Client.Cancel(ctx, runID) 이 이미 있다 — stop 이 먼저 부른다   internal/runctl/client.go:207
   enodectl.exe 심볼 상한은 ci.yml 이 재고 continue-on-error 가 없다     .github/workflows/ci.yml:479~489
     실측 net/http 6 · crypto/tls 1 (상한 50 · 10)
   새 패키지 커버리지 하한 80% 는 프로파일에 나타나는 순간 자동 적용      .github/workflows/ci.yml:269 · decisions §「새 패키지의 커버리지」
```

**추출 경계가 이 유닛의 첫 설계 판단이다.** `proc_*.go` 는 내릴 셋 말고도
`cmdStart`·`cmdStop` 이 쓰는 것을 쥐고 있어, 무엇을 `internal/proc` 로 내리고
무엇을 `cmd/enodectl` 에 남기는지를 FD 에서 파일·줄로 못 박는다. `net/http` 를
안 들이는 것이 이 경계의 목적이다.

---

## 2. 물음 — ADR-068 (진행자 · 착수 블로킹) — 닫힘

**이 유닛의 유일한 미결은 ADR-068 하나였고, 진행자가 A 로 닫았다.** 갈래와
대가를 정리한 별지는
`aidlc-docs/nacl1119/construction/panel/functional-design/adr-068-questions.md`,
캐논 행은 `requirements/decisions.md` §2 「탐지 능력 읽기 (ADR-068) · At 의 출처」다.

```text
   답 A   데몬이 정책 파일 옆 <stem>.status.yaml 에 Caps 와 At 을 쓰고 제어판이 읽는다.
          광고·Mediator 무변경(ADR-068 유지).  형식·자리 세부는 이 FD 가 정한다
```

계약에 걸리는 나머지(바인딩 · LAN 토큰 · stop 순서)는 정본이 이미 값을 정해
두어(decisions §2 · 「LAN 노출의 토큰」 · §6.5) 되묻지 않는다 — 5절에 그대로 딛는다.

---

## 3. Functional Design 실행 계획

각 항목은 산출물의 절 하나에 대응한다.

### 3.1 답 반영

- [x] ADR-068 답을 읽고 어긋남·모호함을 검사한다 — 답 A. decisions §2 「S3 현재 작업의
      출처」와의 긴장을 찾아 진행자가 확인, `decisions.md` §2 에 행을 더해 닫음
- [x] 답을 `aidlc-docs/nacl1119/audit.md` 에 원문 그대로 이어 붙였다 (덮어쓰지 않음)
- [x] 긴장 기록 — 상태 파일 방식이 lease 에서 한 번 물렸으나 At 은 Mediator 에 없어
      같은 논리가 안 닿는다. `adr-068-questions.md` 「닫힘」과 decisions 행에 근거를 적음

### 3.2 domain-entities.md — 제어판이 쓰는 것

- [x] `panel.Config` — `Node · Listen · MediatorBase · Token · PanelToken`. 겉면 시작 모양(정본 §5)
- [x] 화면이 읽는 값 — 신원(node_id · label) · 탐지 능력(Caps + At · 상태 파일에서) ·
      현재 작업(Nodes 원문에서 이 node_id 행의 lease) · Mediator 마지막 응답 시각
- [x] 상태 파일 `<stem>.status.yaml` 의 형 — `caps`(광고 능력 목록과 같은 모양) · `at`(RFC3339).
      `StatusPath(configPath)` 는 `PolicyPath` 와 같은 규칙(<dir>/<stem>.status.yaml)
- [x] drain 상태 모델 — 정책 파일의 `drain` 값 어휘(`""` · `graceful` · `at-boundary`) 를
      그대로 쓴다. 새 형식을 안 만든다 (policy.go 정본)
- [x] `panel_token` 을 `internal/enode` 의 `Policy` 구조체에 additive 로 더하는 자리 —
      정책 파일에 키 하나가 앉을 뿐, policyReader 가 모르는 키를 이미 무시하므로 데몬 무변경
- [x] 프로세스 제어 상태 모델 — status(running · stopped) · start · stop · logs.
      도는 노드에서는 start 자리에 재시작이 온다(stop 뒤 start · 새 하위명령 아님)

### 3.3 business-logic-model.md — 조작의 흐름

- [x] 탐지 능력 읽기 — 데몬이 status 파일을 쓰는 자리(advertise.go 가 정책을 읽는 옆 ·
      snap.At 이 손에 있음)와 제어판이 읽는 경로. 파일이 없거나 낡으면 화면이 무엇을 보이나
- [x] drain 걸기·모드 선택·풀기 — 정책 파일 쓰기(유닉스 0600 · 윈도우 상속 ACL).
      걸기 전 화면(S3)과 건 뒤 화면(S3b)이 서로 다른 장임을 흐름으로 가른다
- [x] stop 의 순서 — POST /v1/runs/{id}/cancel(Client.Cancel) 을 먼저 부르고 데몬을 끈다.
      Mediator 가 안 닿으면 확인 문구가 「임대 만료로 죽는다」를 말하고 그대로 끈다
      (decisions §6.5 표 · verdict 에 cancelled by 가 남게)
- [x] serve 의 exec 위임 흐름 — enodeBin 을 찾아 제어판 하위명령으로 넘긴다(setup.go 본뜸).
      HTTP 서버를 enodectl 안에서 직접 부르지 않는 이유(심볼 상한 · 39배·14배)를 적는다
- [x] 흐름을 ASCII 다이어그램으로 그린다 (`common/ascii-diagram-standards.md`)

### 3.4 business-rules.md — 규칙과 검증

- [x] 바인딩 — 127.0.0.1:8081 기본 (Mediator :8080 옆 · decisions §2 · ADR-063 3.3)
- [x] LAN 노출 — `--listen` 이 127.0.0.1 밖이면 정책 파일의 `panel_token` 필수.
      없으면 기동 거부(`New` 가 error). Mediator 토큰을 재사용하지 않는다. 이 회차엔
      켜지 않고 거부 경로만 선다 (decisions 「제어판 LAN 노출의 토큰」)
- [x] 임포트 경계 — `panel -> store` 금지 · `panel -> api` 금지 · `enode -> panel` 금지
      (`cmd/enode -> panel` 허용). 경계 검사 테스트로 잡는다
- [x] proc 추출 경계 — `internal/proc` 는 net/http 를 안 쓴다. 어느 함수·상수를 내리고
      어느 것을 cmd/enodectl 에 남기는지 파일·줄로 못 박는다 (1절)
- [x] status 파일 쓰기 표면 — `internal/enode` 에 additive 한 자리. 권한은 정책 파일과
      같다(0600 · 상속 ACL). 데몬이 못 써도 막지 않는다(로그만) — 능력은 광고가 이미 진다
- [x] 심볼 상한 — enodectl.exe net/http ≤50 · crypto/tls ≤10. serve 가 exec 위임이라
      제어판 HTTP 표면이 이 실행파일에 안 링크됨을 규칙으로 적는다
- [x] 오류·확인 문구 — stop·재시작은 누르기 전에 그 Run 을 어떻게 끝내는지 말한다

### 3.5 완료 조건 (게이트 CP4)

- [x] `scene-gates.md` CP4(43행 · 90~99행)의 S3·S3b·S1·S5 를 그대로 옮기고 각 요구를 규칙에 잇는다
- [x] **화면 버튼을 전부 나열한다** — 조작 넷(status·start·stop·logs) + drain 걸기·모드
      (graceful·at-boundary)·풀기. 「누르면 걸린다」만으로는 「풀기」가 빠진 채 통과된다
- [x] 경계 검사 테스트 초록 · 심볼 상한 재측정 · `internal/panel` 커버리지 80%
- [x] CP4 는 2일차 정오 · 가장 앞에 당긴 게이트라 늦추지 않는다. CP5·CP6·CP7 은 장면 밖(빨개도 CP4 는 초록)

### 3.6 검토

- [x] 산출물이 `CONVENTIONS.md` 1절(강조는 굵게만 · 장식 문자 금지)·2절(밖으로 나가는 것은 영어)을 지키는지 본다
- [x] `content-validation.md` 로 다이어그램·특수문자를 검사한다
- [x] security-baseline 준수 요약을 낸다 (LAN 토큰 거부 경로 · 정책·상태 파일 권한이 걸리는 규칙)
- [x] 완료 메시지를 2지 선택(변경 요청 / 다음 단계로)으로 냈다. 진행자가 "진행해"(2026-09-09) 로 승인 — NFR Requirements 로 간다. 코드는 아직 안 씀

---

## 4. 낼 파일

```text
   aidlc-docs/nacl1119/construction/panel/functional-design/
     adr-068-questions.md      진행자 질문 · A 로 닫힘 (냄)
     domain-entities.md        제어판이 쓰는 것 (Config · 화면 값 · status 파일 · drain · 프로세스 상태)
     business-logic-model.md   조작의 흐름 (탐지 읽기 · drain · stop 순서 · serve 위임)
     business-rules.md         규칙 · 경계 · 심볼 상한 · 완료 조건
```

화면 세부(레이아웃)는 시안(`design/*.pen`)이 진행자 몫이라, FD 는 화면이 읽고
쓰는 값과 버튼의 존재·의미까지 못 박고 그림은 안 그린다.

---

## 5. 이 유닛이 안 하는 것

```text
   링 파일 · 트랜스크립트 카드        transcript (W4).  같은 담당이지만 다음 유닛이다
   drain 을 광고에 싣는 것            drain (W2 · shin-son) 의 internal/enode 쪽.  panel 은 정책 파일을 쓸 뿐이다
   Capabilities 광고 계약을 바꾸는 것  ADR-068=A 라 안 연다.  광고·Mediator 무변경
   internal/store · internal/api/ui   접점 아님 (핸드오프 7절)
   cmd/mediator · design/ · 루트 상태 파일 둘   진행자 몫
   LAN 토큰을 이 회차에 켜는 것        거부 경로만 선다 (decisions 「LAN 노출의 토큰」)
```

---

## 6. 지금 상태

- 브랜치 `unit/panel` (origin/main 50af6cf 에서 딴 것 · upstream 끊김). 첫 푸시에 `git push -u origin unit/panel`
- 의존 닫힘 — obs(PR #4) · drain(PR #7) main 에 있음
- 착수 블로킹 ADR-068 — A 로 닫힘 (2절 · decisions §2). 이 계획 승인 뒤 FD 산출물을 낸다
- 주의 — 이 세션 초반에 작업 트리가 handoff 커밋으로 리셋되어 한 번 산출물이 사라졌다.
  이 세션이 panel 을 소유하기로 했고(2026-09-09), 리셋 재발을 막기 위해 단계마다 커밋한다
