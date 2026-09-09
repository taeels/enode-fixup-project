# panel — 구현 요약과 CP4 게이트 판정

Code Generation 산출물의 요약이다. 코드는 저장소에 있고(아래 파일), 여기는
무엇을 냈고 게이트가 무엇으로 초록인지를 적는다.

커밋 — `f958ecc`(proc 추출) · `becb42a`(상태 파일) · `d1ca5fc`(제어판 · 배선 · 테스트).

---

## 1. 낸 파일

```text
   internal/proc/                프로세스 프리미티브 (net/http 없음)
     proc.go                     PidFromLock(configPath)
     proc_unix.go                ProcessAlive · SignalStop · SignalKill · OwnsConfig · DetachAttr
     proc_windows.go             같은 표면 (플랫폼 짝)
   internal/enode/status.go      Status{Caps,At} · StatusPath · WriteStatus · ReadStatus (ADR-068=A)
   internal/enode/policy.go      Policy 에 panel_token · ReadPolicyFile · WritePolicyFile
   internal/enode/advertise.go   snap := a.Caps() 뒤 At 바뀔 때만 WriteStatus (guarded)
   internal/panel/               제어판 서버
     panel.go                    Config · New · Handler · isLoopback · requireToken
     view.go                     State 뷰 모델 · thisNode(두 홉) · fillStep
     handlers.go                 state · drain · undrain · stop(cancel 먼저) · start · logs
     page.go                     기능판 HTML (버튼 전부 · 꾸밈은 시안 몫)
   cmd/enode/panel.go            enode panel 하위명령
   cmd/enodectl/serve.go         enodectl serve <name> — enode panel 로 exec 위임
```

만진 파일 — `cmd/enodectl/main.go`(pidOf -> proc.PidFromLock · detachAttr ->
proc.DetachAttr · serve 등록) · `caffeinate_darwin.go`(proc.DetachAttr) ·
`cmd/enode/main.go`(panel 분기).

---

## 2. CP4 게이트 판정 (scene-gates CP4 · §2.1)

**화면 버튼을 전부 낸다** (business-rules §6 의 표).

```text
   조작        내는 자리                         근거
   ─────────   ──────────────────────────────   ────────────────────────────
   status      refresh + 프로세스 배지            handleState · page.go
   start       멈춘 노드 (S5)                     handleStart · page.go
   재시작       도는 노드 (stop 뒤 start)          page.go (새 하위명령 아님)
   stop        도는 노드 · cancel 먼저            handleStop (Cancel -> SignalStop)
   logs        언제나                            handleLogs
   drain 걸기   안 걸린 노드 (S3) · 모드 선택       handleDrain (graceful · at-boundary)
   drain 풀기   걸린 노드 (S3b)                    handleUndrain
```

S3(걸기 전)과 S3b(건 뒤)를 다른 장으로 그린다(page.go 가 drain 값으로 가름).
stop·재시작은 누르기 전에 cancel-먼저를 확인 문구로 말한다.

**게이트 재료 재측정.**

```text
   심볼 상한       enodectl.exe  net/http=13 (<=50) · crypto/tls=1 (<=10)   초록
   커버리지        internal/panel 87.8% · internal/proc 88.9% (>=80)        초록
   임포트 경계     panel 은 store/api 안 딛음 · enode 는 panel 안 딛음       초록 (boundary_test)
   크로스 빌드     linux · darwin · windows go build ./...                   초록
   vet · glyphscan · gofmt                                                  초록
```

**보류 하나 (게이트 밖) — scene-gates §4.** 화면의 눈 검증(S3/S3b/S1/S5 를
실제 데몬·브라우저로 보는 것)과 LED 하드웨어 확인은 사람+하드웨어가 닫는다.
이 유닛의 코드·자동 검사는 초록이고, 눈 검증은 진행자·하드웨어가 CP4 를 최종으로
닫을 때 한다. 코드 게이트 재료는 전부 초록이다.

---

## 3. 정본과 어긋난 자리 (기록)

`enode-features.md` 3.1.2 는 proc 함수 셋을 `internal/panel` 로 내리라 했으나,
그러면 `cmd/enodectl` 이 그 패키지를 거쳐 net/http 에 닿아 심볼 상한이 깨진다.
유닛 정본 `unit-of-work.md` §5(`internal/proc` · net/http 없음)가 이겨 그것으로
갔다. (business-rules §3 · domain-entities §8)

---

## 4. 접점 (진행자 직렬 병합)

`internal/enode/advertise.go` — drain(shin-son · 병합됨)이 만진 파일. panel 이
guarded 호출 하나를 더했다. 나머지는 신규 파일이거나 nacl1119 소유
(`internal/panel` · `internal/proc` · `cmd/enode/panel.go` · `cmd/enodectl/serve.go`).
