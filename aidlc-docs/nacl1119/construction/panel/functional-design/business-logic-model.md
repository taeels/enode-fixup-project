# panel — 업무 흐름 (business-logic-model)

제어판이 하는 일의 흐름. 읽기 넷(신원·탐지 능력·현재 작업·drain 상태)과 쓰기
셋(drain 토글·프로세스 제어·상태 파일은 데몬이 쓴다)이다. 정본은
`enode-features.md` 3.1.2 · `decisions.md` §2 · §「제어판 서버 위치」 다.

---

## 1. serve 의 exec 위임 — 왜 프로세스를 가르나

```text
   enodectl serve <name>
     -> oneName(name) 으로 설정 경로를 푼다
     -> enodeBin() 을 찾는다 (cmd/enodectl/setup.go 의 cmdSetup 과 같은 규칙)
     -> exec.Command(enodeBin, "panel", "--config", <경로>, "--listen", ...) 로 넘긴다
     -> 넘긴 쪽(enode)의 종료 코드를 그대로 되쓴다 (setup.go:39 본뜸)

   enode panel  (cmd/enode 의 새 하위명령 · os.Args[1]=="panel" 분기)
     -> panel.New(Config{...})
     -> http.ListenAndServe(cfg.Listen, srv.Handler())
```

**HTTP 서버를 `enodectl` 안에서 직접 부르지 않는다.** `cmdSetup` 이 이미 같은
이유로 같은 모양이다(`setup.go`) — `enode.SetupCLI` 를 `enodectl` 에서 부르면
링커가 `net/http`·`crypto/tls` 까지 끌어와 `enodectl.exe` 가 백신에 지워진
사건이 있었다(앞 회차 실측 심볼 39배·14배). `serve` 를 exec 위임으로 두면
제어판의 HTTP 표면이 `enodectl.exe` 링크 그래프에 안 들어온다(CP4 심볼 상한).

```text
   [enodectl serve]  --exec-->  [enode panel]  --net/http-->  [internal/panel]
    net/http 없음                 net/http 있음 (데몬 실행파일엔 이미 있다)
    proc 만 링크
```

---

## 2. 탐지 능력 읽기 (데몬이 쓰고 제어판이 읽는다)

```text
   데몬 쪽 (cmd/enode · internal/enode)
     advertise 루프가 매 주기 snap := a.Caps() 로 Capabilities{Caps, At} 를 든다
     (advertise.go:159 · 이미 손에 있다)
     At 이 지난 쓰기와 다르면 enode.WriteStatus(configPath, snap) 을 부른다
     -> <stem>.status.yaml 에 caps · at 을 쓴다 (0600 · 상속 ACL)
     못 써도 막지 않는다 — 능력은 광고가 이미 진다.  경고만 (business-rules §5)

   제어판 쪽 (internal/panel)
     enode.ReadStatus(configPath) -> Status{Caps, At}
     화면에 caps 를 읽기 전용으로, at 을 「언제 잰 값인가」로 보인다 (features 3.1.2)
     파일이 없거나 낡으면 「아직 모름」 (business-rules §2)
```

At 이 바뀔 때만 쓰는 이유 — 광고 60초 대 탐지 5분이라 매 광고 쓰기는 같은
값의 낭비다(`main.go:67,76`).

---

## 3. stop 의 순서 — cancel 먼저

`enode-features.md` 3.1.2 수용 기준 · `decisions.md` §「제어판」(6.5/7.2)이 정본.

```text
   stop (또는 도는 노드의 재시작)
     1. lease 가 있나?  runctl.Client.Nodes 로 이 노드의 lease.run_id 를 본다
     2. 있으면 runctl.Client.Cancel(ctx, run_id) 를 먼저 부른다
        -> Run 이 취소 경로로 종료하고 verdict 에 cancelled by 가 남는다
     3. 그다음 proc.SignalStop(pid) 로 데몬을 끈다
     4. Mediator 가 안 닿아 2 가 실패하면 -> 확인 문구가
        「임대 만료로 죽는다(lease expired: renewal stopped) — 소유자가 껐다는
        기록이 안 남는다」를 말하고, 사람이 확인하면 그대로 3 을 한다
```

**왜 cancel 이 먼저인가** — 안 그러면 그 Run 이 `lease expired: renewal
stopped` 로 죽어 소유자가 껐다는 사실이 기록에 안 남고, drain 이 `verdict` 에
`drain:<node_id>` 를 남기는 것과 대칭이 깨진다(features 3.1.2). 재시작도
같은 규칙이다 — 새 하위명령이 아니라 stop 뒤 start 다.

---

## 4. drain 토글 — 정책 파일 쓰기

```text
   걸기      정책 파일에 drain: graceful (또는 at-boundary) 를 쓴다.
             데몬이 다음 광고 직전에 그 파일을 읽어 광고에 싣는다 (policy.go · 캐시 없음)
   모드      graceful | at-boundary.  기본은 graceful (decisions §1)
   풀기      정책 파일에 drain: "" 를 쓴다.  자동 복귀 없음 — 소유자가 명시로 푼다
   권한      쓸 때 0600 (유닉스) · 상속 ACL (윈도우).  policy.go 의 checkPerms 와 같은 기준
```

화면은 걸기 전(S3)과 건 뒤(S3b)를 다른 장으로 그린다 — 한 장에 둘 다 못
그린다(scene-gates §2.1). 건 뒤에는 현재 모드 배지 · 「지금 도는 작업 없음」 ·
「풀기」가 보인다.

---

## 5. 프로세스 상태 · Mediator 연결 상태

```text
   status   proc.ProcessAlive(pidOf(name)).  running | stopped
   start    멈춘 노드를 detachAttr 로 띄운다 (cmd/enodectl 쪽 · 제어판은 그 경로를 부른다)
   logs     데몬 로그를 보인다 (트랜스크립트 카드와 다른 물건 — 그건 transcript 유닛)
   Mediator 연결   제어판이 자기 runctl 호출의 마지막 성공 시각을 든다.
                   끊기면 drain 통보가 늦어진다는 문구를 보인다 (features 3.1.2 · S1b 의 짝)
```

---

## 6. 흐름 한눈 (ASCII)

```text
                         +---------------------------+
                         |   internal/panel (server) |
                         |   127.0.0.1:8081          |
                         +------------+--------------+
                                      |
        +-----------------+-----------+-----------+------------------+
        |                 |                       |                  |
   [로컬 파일]        [로컬 proc]            [Mediator 조회]      [데몬이 씀]
        |                 |                       |                  |
  policy.yaml        ProcessAlive           runctl.Client       status.yaml
  (drain·panel_token) SignalStop            .Nodes/.Status      (caps·at)
        |             OwnsConfig             .Cancel                 ^
        v                 |                       |                  |
  drain 토글          status/start/         현재 작업 · instance      |
  (걸기/모드/풀기)      stop/logs            (두 홉)              advertise 루프가
                                                                  At 바뀔 때 씀
```

제어판은 무상태다 — 위 넷을 그때그때 읽어 그린다. 데몬이 죽어도 로컬 파일과
proc 는 답하고, Mediator 가 끊겨도 로컬 묶음은 산다.
