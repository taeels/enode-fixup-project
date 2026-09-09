# panel — NFR Design

NFR Requirements 의 요구를 **배선**으로 확정한다. 가장 무거운 하나는 심볼
상한(CP4 차단)이고, 그것이 패키지 경계와 exec 위임의 모양을 정한다. 정본 —
NFR Requirements(같은 폴더) · `constraints.md` · `decisions.md` §2 ·
`cmd/enodectl/setup.go` · `internal/enode/advertise.go`.

---

## 1. 심볼 상한 회피 — 링크 그래프 설계

**규칙 — `cmd/enodectl` 의 링크 그래프가 `net/http`·`crypto/tls` 에 짙게 닿지
않게 한다.** `enodectl.exe` 만 상한이 걸린다(`ci.yml:479`).

```text
   실행파일        임포트하는 것                         net/http 링크
   ─────────────   ──────────────────────────────────   ──────────────────────
   enodectl        internal/proc · os/exec (serve) ·      얕다.  지금 6 · crypto/tls 1
                   기존 cmd/enodectl 코드                 상한 50 · 10 안
   enode (데몬)     internal/panel · internal/enode ·     짙다.  그러나 상한 대상이 아니다
                   net/http                              (데몬은 원래 net/http 를 쓴다)
```

두 배선이 이 규칙을 집행한다.

```text
   serve 는 exec 위임    enodectl serve <name> 이 internal/panel 을 임포트하지 않는다.
                        os/exec 로 enode panel 을 띄운다 (setup.go 의 cmdSetup 과 같은 모양).
                        그래서 제어판의 HTTP·TLS 표면이 enodectl.exe 에 안 링크된다

   proc 는 net/http 없음   internal/proc 는 syscall·os/exec 만 쓴다.  enodectl 이 stop·start 로
                          이 패키지를 임포트해도 상한이 안 는다.  panel 로 안 내리는 이유가 이것
                          (panel 은 net/http 서버라 enodectl 이 그리로 닿으면 상한이 깨진다)
```

**검증** — 유닛 종료 시 `go tool nm /tmp/enodectl.exe | grep -c ' T net/http\.'`
와 `crypto/tls` 를 재측정해 상한 안임을 완료 조건에 적는다(business-rules §4).

---

## 2. 상태 파일 쓰기 — 자리와 트리거

```text
   자리      internal/enode/status.go(신규)  StatusPath · WriteStatus · ReadStatus
   호출      advertise 루프가 이미 snap := a.Caps() 로 Capabilities{Caps, At} 를 드는
            자리(advertise.go:159) 바로 뒤.  정책 파일을 읽는 곳(readPolicy)과 같은 리듬
   트리거    snap.At 이 지난 쓰기와 다를 때만 (advertiser 가 lastStatusAt 을 기억)
   원자성    <stem>.status.yaml.tmp 에 쓰고 os.Rename 으로 바꾼다 — 제어판이 반쯤 쓴
            파일을 읽지 않게.  (윈도우 rename 은 대상이 열려 있으면 실패할 수 있으나
            제어판은 읽고 곧 닫으므로 다음 주기에 다시 쓴다 — At 트리거라 자주 안 쓴다)
   권한      0600 (유닉스).  윈도우는 상속 ACL — policy.go 가 윈도우 권한을 안 거는 것과 같다
   실패      WriteStatus 오류는 advertiser 가 경고만 찍고 넘어간다.  같은 원인이 이어지면
            다시 안 찍는다(policy.go 의 warn 규약을 본뜬다).  능력은 광고가 이미 진다
```

접점 — `advertise.go` 에 더하는 것은 guarded 호출 하나이고 로직은 `status.go`
에 산다. drain(shin-son)이 이 파일을 이미 만졌으므로 진행자 직렬 병합 대상으로
표시한다(`CONVENTIONS.md` 3.5).

---

## 3. 신뢰성 — 저하를 배선으로

제어판의 각 읽기가 독립 실패한다. 하나가 막혀도 화면 나머지가 선다.

```text
   runctl 호출 타임아웃   Mediator 조회에 짧은 타임아웃(예: 3초)을 건다 — 불통이 화면을
                        멈추지 않게.  실패는 「Mediator 마지막 응답 N초 전」으로 잡는다
   읽기 격리            신원·탐지 능력·drain·프로세스는 로컬이라 Mediator 와 무관하게 그린다
   부분 렌더            한 묶음의 오류가 예외로 전체를 비우지 않는다 — 그 카드만 상태 문구를 낸다
```

`runctl.Client.HTTP` 에 타임아웃을 건 클라이언트를 제어판이 만들어 넘긴다
(데몬의 롱폴 클라이언트와 다르다 — 제어판은 롱폴을 안 쓴다).

---

## 4. 보안 배선

```text
   isLoopback(host)     New(cfg) 가 Listen 의 호스트를 판정한다 — 127.0.0.1·localhost·::1
                        이 아니면 LAN.  LAN 인데 PanelToken 이 비면 error 를 내고 안 뜬다
   panel_token 출처      정책 파일에서 읽어 Config.PanelToken 에 넣는다.  화면·응답에 안 싣는다
   파일 권한            정책·상태 파일을 쓸 때 0600(유닉스).  느슨하면 경고 (checkPerms 규약)
   읽기 전용 카드        탐지 능력 카드는 쓰기 핸들러가 없다 — GET 만.  편집 자리 0
```

---

## 5. 시험성 배선 (커버리지 80% · DB 없이)

```text
   internal/panel   httptest.Server 로 가짜 Mediator 를 세워 Nodes·Status·Cancel 응답을 준다.
                    정책·상태 파일은 t.TempDir() 에 만든다.  Postgres 를 안 탄다
   internal/proc    플랫폼 짝이라 각 OS 빌드에서 자기 태그만 돈다.  ProcessAlive 는 자기
                    프로세스(os.Getpid)로, OwnsConfig 는 임시 경로로 검사
   경계 테스트       go/packages 또는 import 목록 단언으로 panel->store/api · enode->panel 금지를 잡는다
   stop 순서        가짜 Mediator 에 Cancel 이 먼저 오는지 · 불통 시 경고 경로를 단언한다
```

---

## 6. 이식성 배선

```text
   빌드 태그 짝     proc_unix.go(!windows) · proc_windows.go(windows)를 internal/proc 로 옮긴다.
                  함수 시그니처가 두 파일에서 같아야 한다 (ProcessAlive·SignalStop·OwnsConfig)
   OS 권한         상태·정책 파일 권한 분기를 한 곳(status.go·policy.go)에 둔다
   크로스 빌드      linux·darwin·windows 셋 다 go build ./... 통과 (ci.yml cross 잡)
```

---

## 7. 이 설계가 넓히지 않는 것

```text
   새 Go 의존       0.  net/http·yaml 은 이미 있다.  go.mod·packaging 무변경
   Capabilities 광고   안 만진다.  At 은 상태 파일로만 나간다 (ADR-068=A)
   데몬의 서버화     안 한다.  데몬은 여전히 나가는 http.Client 만.  상태 파일이 통로다
```
