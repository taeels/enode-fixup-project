# panel — 규칙 · 검증 · 완료 조건 (business-rules)

정본 — `enode-features.md` 3.1.2 · `requirements/scene-gates.md` CP4(43행 ·
90~99행) · `decisions.md` §2 · §「제어판 서버 위치」·「LAN 노출의 토큰」 ·
`constraints.md`(구조 불변식 · 심볼 상한).

---

## 1. 바인딩과 LAN 노출

```text
   바인딩 기본   127.0.0.1:8081 (Mediator :8080 옆 · ADR-063 §3.3 · decisions §2)
   LAN 노출      --listen 이 127.0.0.1 밖일 때만.  그때 정책 파일의 panel_token 이 필수
   거부          panel_token 이 비면 New(cfg) 가 error 를 내고 serve 가 안 뜬다
   토큰 재사용   Mediator 토큰을 안 쓴다 — 함대 전체의 신뢰 경계라 기계 하나에 안 흘린다
   이 회차       LAN 을 안 켠다.  거부 경로만 선다 (decisions 「LAN 노출의 토큰」)
   기본 무인증   127.0.0.1 은 그 기계에 접속한 것이 소유의 증거다 (ADR-063 §3)
```

`New` 의 판정 — `Listen` 의 호스트가 `127.0.0.1`·`localhost`·`::1` 이 아니면
LAN 으로 보고 `PanelToken` 빈 값을 error 로 막는다.

---

## 2. 값이 없을 때의 화면 (분기)

읽는 값마다 「없음」이 다른 뜻이라 화면이 달라야 한다.

```text
   탐지 능력    상태 파일 없음/못 읽음  -> 「아직 모름」 (데몬이 첫 광고 전이거나 못 씀)
   현재 작업    이 노드가 Mediator 에 없음 -> 멈췄거나 광고 만료.  프로세스 status 로 가른다
               lease 없음               -> 「지금 도는 작업 없음」
               Mediator 불통            -> 로컬 묶음은 그대로, 「Mediator 마지막 응답 N초 전」
   프로세스     pid 없음/죽음            -> stopped.  start(S5)가 보인다
   drain       정책 파일 없음           -> 안 걸림 (policy.go 와 같은 접기)
```

빈 값을 「0」이나 「없음」 한 글자로 뭉치지 않는다 — 멈춘 노드와 Mediator 불통은
소유자가 할 일이 다르다.

---

## 3. 임포트 경계와 proc 추출

```text
   금지   panel -> store · panel -> api · enode(라이브러리) -> panel
   허용   cmd/enode -> panel (제어판 하위명령) · panel -> enode(라이브러리 · Derive·Status·정책)
   검사   internal/panel 을 처음 만드는 유닛이 임포트 경계 검사 테스트를 낸다
```

**proc 는 net/http 를 안 쓴다.** `ProcessAlive`·`SignalStop`·`OwnsConfig` 를
`internal/proc` 로 내린다(빌드 태그 짝 그대로). `start` 가 쓰는
`detachAttr`·`signalKill`·`exeSuffix` 는 `cmd/enodectl` 에 남는다.

**정본 충돌을 여기 못 박는다** — `enode-features.md` 3.1.2(:198)는 이 셋을
`internal/panel` 로 내리라 적었다. 그러나 `cmd/enodectl` 의 `stop`·`start` 가
이 셋을 부르므로, `internal/panel`(net/http 서버)에 두면 `enodectl.exe` 링크
그래프가 그 패키지를 거쳐 `net/http` 에 닿아 심볼 상한(4절)이 깨진다. 유닛
정본 `unit-of-work.md` §5 는 `internal/proc`(net/http 없음)로 내리라 적었고,
그것이 상한을 지키는 유일한 갈래다. canon 서열대로 유닛 정본이 이긴다 —
`internal/proc` 로 간다.

---

## 4. 심볼 상한 (CP4 차단 게이트)

```text
   대상    enodectl.exe (GOOS=windows).  ci.yml:479~489 · continue-on-error 없음
   상한    net/http T 심볼 <= 50 · crypto/tls T 심볼 <= 10
   지금값  6 · 1 (실측)
   규칙    serve 는 exec 위임이라 제어판 HTTP 표면이 이 실행파일에 안 링크된다.
           proc 가 net/http 를 안 쓴다.  이 둘이 상한을 지키는 방법이다
   완료    유닛을 닫을 때 재측정해 상한 안임을 확인한다 (features 3.1.2 비기능)
```

---

## 5. 상태 파일 쓰기 규칙 (데몬 쪽)

```text
   자리     internal/enode.  advertise 루프가 이미 정책을 읽는 옆에서
            enode.WriteStatus(configPath, snap) 을 부른다 (advertise.go 접점 · 한 호출)
   조건     snap.At 이 지난 쓰기와 다를 때만 쓴다
   권한     0600 (유닉스) · 상속 ACL (윈도우).  정책 파일과 같다
   실패     데몬이 못 써도 막지 않는다 — 능력은 광고가 이미 진다.  경고만 찍고,
            같은 원인이 이어지면 다시 안 찍는다 (policy.go 의 warn 규약을 본뜬다)
```

접점 표시 — `advertise.go` 는 drain(shin-son)이 병합한 파일이다. panel 이
더하는 것은 guarded 한 호출 하나이고 새 자리는 `internal/enode/status.go` 다.
`CONVENTIONS.md` 3.5 의 「행렬 밖 파일을 만진 diff」로 가져간다.

---

## 6. 완료 조건 — 게이트 CP4

`scene-gates.md` CP4(43행) · §2.1 화면 검증이 정본이다. **화면의 버튼을 전부
나열한다** — 「누르면 걸린다」만으로는 「풀기」가 빠진 채 통과된다(§2.1).

```text
   S3    걸기 전 — 「걸기」 버튼 · 모드 선택(graceful · at-boundary) ·
         Mediator 마지막 응답 시각
   S3b   건 뒤 — 현재 모드 배지 · 「지금 도는 작업 없음」 · 「풀기」 버튼
         걸기 전과 건 뒤는 서로 다른 장이다 (한 장에 둘 다 못 그린다)
         조작 넷이 전부 있다 — status · start · stop · logs
         도는 노드에서는 start 자리에 재시작이 보인다 (stop 뒤 start · 새 하위명령 아님)
         stop · 재시작은 누르기 전에 그 Run 을 어떻게 끝내는지 말한다 (cancel 먼저)
   S1    배지가 「draining · 진행 중」과 「draining · 대기 중」으로 갈린다
   S5    멈춘 노드에서 start 가 보이고 누르면 뜬다
```

조작을 세는 표(빠짐 방지).

```text
   조작        보이는 자리        누르기 전 문구
   ─────────   ───────────────    ──────────────────────────────
   status      언제나
   start       멈춘 노드 (S5)
   재시작       도는 노드          그 Run 을 cancel 먼저 (stop 과 같은 규칙)
   stop        도는 노드          그 Run 을 cancel 먼저 · Mediator 불통 시 임대 만료 경고
   logs        언제나
   drain 걸기   안 걸린 노드 (S3)  모드 선택과 함께
   drain 풀기   걸린 노드 (S3b)    현재 모드 배지와 함께
```

더해 — 경계 검사 테스트 초록 · 심볼 상한 재측정(4절) · `internal/panel`
커버리지 80%(DB 없이 도는 테스트로 채운다 · decisions §「새 패키지의 커버리지」).

**CP4 는 2일차 정오 · 이 팩이 가장 앞에 당긴 게이트다.** CP5·CP6·CP7 은 장면
밖이라 빨개도 CP4 는 초록이다(scene-gates).

---

## 7. security-baseline 준수 요약

`decisions.md` §3 의 처리를 이 유닛에 적용한 결과다.

```text
   규칙          이 유닛에서                                          판정
   ───────────   ─────────────────────────────────────────────────   ──────────────
   SECURITY-01   평문 HTTP.  기존 표면과 같다 (constraints §6)          기록만 (N/A)
   SECURITY-07   새 표면이 127.0.0.1 기본.  LAN 은 --listen 명시로만    적용 (거부 경로)
   SECURITY-08   접근 제어 — 그 기계 접속이 소유 증거(ADR-063 §3).      적용
                 LAN 노출 시 panel_token 없으면 기동 거부
   SECURITY-12   인증 — Mediator 토큰 재사용 안 함.  panel_token 은     적용
                 정책 파일에만(0600).  브라우저에 실 토큰 없음
   파일 권한      정책·상태 파일 유닉스 0600 · 윈도우 상속 ACL.          적용
                 남이 쓸 수 있으면 경고(policy.go checkPerms 규약)
```

읽기 전용 카드(탐지 능력)는 편집 자리를 안 만든다 — 광고가 매번 전부이고
다음 광고가 덮는다(ADR-017 결정 3).

---

## 8. 이 유닛이 안 하는 것

```text
   Capabilities 광고 계약 변경   ADR-068=A 라 안 연다.  광고·Mediator·schema 무변경
   트랜스크립트 카드 · 링 파일    transcript (W4).  logs 카드와 다른 물건이다
   drain 을 광고에 싣기          drain (W2 · shin-son) 의 internal/enode 쪽
   LAN 토큰을 이 회차에 켜기      거부 경로만 선다
   화면 레이아웃(그림)           시안 design/*.pen — 진행자 몫.  FD 는 값과 버튼까지만
```
