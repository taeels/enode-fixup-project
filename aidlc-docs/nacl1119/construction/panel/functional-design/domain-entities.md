# panel — 자료 모델 (domain-entities)

제어판이 쓰는 것. 새 DB 표가 없다 — 제어판은 무상태이고, 값은 셋에서 온다:
로컬 설정·파일(신원·정책·상태 파일), Mediator 조회(runctl.Client), 로컬
프로세스(proc). 정본은 `enode-features.md` 3.1.2 · `unit-of-work.md` §5 ·
`decisions.md` §2 「탐지 능력 읽기(ADR-068)」 다.

---

## 1. panel.Config — 서버의 입력

`unit-of-work.md` §5 가 겉면을 얼렸다(`{Node, Listen, MediatorBase, Token,
PanelToken string; ...}`). `...` 자리에 `ConfigPath` 를 더한다 — 정책 파일과
상태 파일이 그 옆에 살아, 경로 없이는 drain 토글도 탐지 능력 읽기도 못 한다.

```go
type Config struct {
    Node         string // 노드 이름 (표시 · serve <name> 가 넘긴다)
    ConfigPath   string // 이 노드의 설정 파일 경로.  정책·상태 파일이 그 옆이다
    Listen       string // 바인딩.  기본 127.0.0.1:8081
    MediatorBase string // runctl.Client.Base
    Token        string // runctl.Client.Token
    PanelToken   string // LAN 노출 시 필수 (정책 파일에서 읽어 넣는다)
}

func New(cfg Config) (*Server, error) // Listen 이 127.0.0.1 밖인데 PanelToken 비면 error
func (s *Server) Handler() http.Handler
```

`ConfigPath` 의 출처 — `enodectl serve <name>` 이 기존 `oneName(name)` 규칙으로
이름을 설정 경로로 풀고(`cmd/enodectl/main.go` 의 `pidOf`·`identityOf` 와 같은
규칙) 그 경로를 제어판 프로세스에 넘긴다. 이름 생략은 없다 (3.1.2 「위치」).

---

## 2. 화면이 읽는 값 (view model)

화면은 다섯 묶음을 그린다. 각 묶음의 출처가 다르다 — 그래서 하나가 죽어도
나머지는 산다(예: Mediator 가 안 닿아도 신원·탐지 능력·프로세스 상태는 로컬이다).

```text
   묶음            필드                                         출처
   ────────────    ─────────────────────────────────────────   ─────────────────────────────
   신원            node_id · label · principal                  로컬 enode.Derive(ConfigPath)
                   instance                                     Mediator 조회 (이 노드의 광고 행) ·
                                                                멈춘 노드는 없음
   탐지 능력        caps[] (읽기 전용) · at (마지막 탐지 시각)     로컬 상태 파일 <stem>.status.yaml
   현재 작업        lease{run_id, not_after} ·                   Mediator 조회 (두 홉 · 3절)
                   CLAIMED 단계 이름 · attempt · started_at
   drain           mode ("" · graceful · at-boundary)           로컬 정책 파일 <stem>.policy.yaml
   프로세스         status(running · stopped) · pid              로컬 proc (processAlive · pidOf)
   Mediator 연결    마지막 응답 시각                              제어판이 자기 runctl 호출에서 잰다
```

**신원의 principal 을 로컬에서 낸다** — `constraints.md` §4 가 `GET /v1/nodes`
응답에서 principal 을 뺐다(ADR-015 §1). 제어판은 자기 기계라 `enode.Derive`
로 직접 낸다. instance 는 데몬이 이번 생마다 새로 뽑는 값이라(ADR-030) 로컬
설정에 없다 — Mediator 광고 행의 `instance` 를 읽고, 멈춘 노드는 빈다.

---

## 3. 현재 작업 — 두 홉 (Mediator 조회)

`decisions.md` §2 「S3 현재 작업의 출처」가 정본이다. 데몬 로그를 안 읽는다 —
관측 기준을 하나로 둔다(ADR-065 §3).

```text
   홉 1   runctl.Client.Nodes(ctx)  ->  원문 JSON.  이 node_id 행의 lease{run_id, not_after}
   홉 2   lease.run_id 가 있으면 runctl.Client.Status(ctx, run_id)  ->  Run.Steps 에서
          state=="CLAIMED" 단계의 ID · attempt · started_at
```

`runctl.Client.Nodes` 는 원문 `json.RawMessage` 를 낸다 — 제어판이 자기
`node_id` 행만 뽑아 파싱한다(파싱 타입은 panel 몫 · obs 가 그렇게 남겼다).
`Step.StartedAt` 은 obs(W0)가 이 유닛을 위해 미리 세운 필드다
(`client.go:127`) — panel 이 되열 필요가 없다.

멈춘 노드 · 임대 없음 · Mediator 불통은 각각 다른 화면이다 (business-rules §2).

---

## 4. 상태 파일 <stem>.status.yaml (ADR-068 = A · 신규)

데몬이 쓰고 제어판이 읽는다. 광고·Mediator 는 안 바뀐다.

```text
   경로     StatusPath(configPath) = <dir>/<stem>.status.yaml
            PolicyPath 와 같은 규칙 (policy.go:32 을 본뜬다)
   형식     caps: [contract.Capability 와 같은 모양]   # 광고에 싣는 것과 같은 값
            at:   RFC3339 (마지막으로 알아낸 시각)
   쓰기     데몬.  advertise 직전에 snap := a.Caps() 로 이미 손에 있는
            Capabilities{Caps, At} 를 쓴다.  At 이 지난 쓰기와 다를 때만 쓴다
            (광고는 60초, 탐지는 5분이라 매 광고 쓰기는 낭비다)
   읽기     제어판.  없거나 못 읽으면 「아직 모름」 (business-rules §2)
   권한     유닉스 0600 · 윈도우는 그 디렉터리의 상속 ACL (정책 파일과 같다)
```

Go 표면(신규 `internal/enode/status.go`).

```go
type Status struct {
    Caps []contract.Capability `yaml:"caps"`
    At   time.Time             `yaml:"at"`
}
func StatusPath(configPath string) string          // <dir>/<stem>.status.yaml
func WriteStatus(configPath string, c Capabilities) error // 데몬이 부른다
func ReadStatus(configPath string) (Status, error) // 제어판이 부른다
```

**왜 상태 파일인가** — `At` 은 데몬 메모리(`detector.go:31` 의 `Capabilities`)
에만 있고 매처가 안 써 광고로 안 나간다(ADR-068). `seen_at`(광고 시각)은 탐지
시각과 최대 5분 어긋난다. 같은 기계의 제어판↔데몬 통로를 이 저장소는 이미
정책 파일로 정해 뒀다(ADR-063) — 상태 파일은 그 방향만 뒤집은 것이라 새 개념이
없다. (§2 「S3 현재 작업의 출처」가 상태 파일을 물린 것은 lease 에 한정된다 —
lease 는 Mediator 가 이미 들지만 `At` 은 안 든다.)

---

## 5. 정책 파일에 panel_token 을 더한다 (additive)

drain 토글이 쓰는 정책 파일(`policy.go` 정본)에 키 하나가 더 앉는다. 새 형식을
안 만든다.

```go
// internal/enode/policy.go
type Policy struct {
    Drain      string `yaml:"drain"`
    PanelToken string `yaml:"panel_token"` // 더한다.  데몬은 안 쓴다 (읽기만 additive)
}
```

데몬의 `policyReader.Read()` 는 `Drain` 만 보고 모르는 키를 이미 무시하므로
(`policy.go:58`), 이 키를 더해도 데몬 동작은 안 바뀐다. 제어판은 LAN 노출을
켤 때 이 값을 읽어 `Config.PanelToken` 에 넣는다 (이 회차엔 거부 경로만 · §1).

---

## 6. drain 상태 어휘

정책 파일의 `drain` 값을 그대로 쓴다 (`contract` 의 상수 · policy.go:74).

```text
   ""            안 걸림
   graceful      도는 단계까지.  기본 (decisions §1)
   at-boundary   단계 경계에서 임대를 놓고 취소 경로로 종료 (decisions §1)
```

화면은 「걸기 전」(S3)과 「건 뒤」(S3b)를 다른 장으로 그린다 — 건 뒤에는 현재
모드 배지와 「풀기」가 보인다. 「풀기」는 정책 파일에 `drain: ""` 를 쓰는 것이다
(자동 복귀 없음 · decisions §1 「drain 해제」).

---

## 7. 프로세스 제어 상태

`enodectl` 의 조작 넷을 감싼다. 새 하위명령을 안 만든다.

```text
   status   proc.ProcessAlive(pid) · pid 는 잠금 파일에서 (pidOf)
   start    멈춘 노드를 띄운다.  detachAttr 로 떼어낸다 (cmd/enodectl 에 남는다 · 8절)
   stop     도는 Run 을 먼저 cancel 하고 데몬을 끈다 (business-logic-model §3)
   logs     데몬 로그를 보인다
   재시작    새 하위명령이 아니다 — 도는 노드 화면에서 start 자리에 보일 뿐.
            stop 뒤 start 이고 stop 과 같은 cancel-먼저 규칙을 따른다 (features 3.1.2)
```

---

## 8. proc 추출 경계 (내리는 것 · 남기는 것)

`internal/proc` 는 `net/http` 를 안 쓴다 — 그것이 이 패키지를 가르는 이유다
(business-rules §3 · 심볼 상한).

```text
   internal/proc 로 내린다     ProcessAlive(pid) · SignalStop(pid) · OwnsConfig(pid, conf)
                              빌드 태그 짝(proc_unix.go · proc_windows.go)을 그대로 옮긴다.
                              제어판과 cmd/enodectl 둘이 같이 쓴다

   cmd/enodectl 에 남긴다     detachAttr · signalKill · exeSuffix · (윈도우)프로세스 생성 상수.
                              start 가 프로세스를 띄우는 자리라 여기 산다.  net/http 와 무관
```

**왜 갈리나** — `enode-features.md` 3.1.2(:198)는 이 셋을 `internal/panel` 로
내리라 적었으나, 그러면 `cmd/enodectl` 의 `stop`·`start` 가 `internal/panel`
을 거쳐 `net/http` 에 닿아 `enodectl.exe` 심볼 상한(CP4)이 깨진다. 유닛 정본
`unit-of-work.md` §5 는 `internal/proc`(net/http 없음)로 내리라 적었고, 그것이
상한을 지키는 유일한 갈래다. **정본이 어긋나면 유닛 정본이 이긴다** —
`internal/proc` 로 간다(business-rules §3 에 근거를 다시 적는다).
