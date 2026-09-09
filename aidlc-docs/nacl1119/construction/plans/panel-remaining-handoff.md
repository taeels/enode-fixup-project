# panel/transcript 남은 작업 인계 요약

작성 2026-09-09 · 담당 handle **nacl1119** · panel Functional Design 승인 직후.
정본과 어긋나면 정본이 이긴다.

---

## 1. 지금까지 (unit/panel 에 커밋됨)

- 브랜치 `unit/panel` — `origin/main`(50af6cf)에서 딴 것, **upstream 끊겨 있음**.
  첫 푸시는 `git push -u origin unit/panel`.
- 커밋 — `04aa712`(FD 계획+ADR-068 결정) · `64b5ae2`(FD 산출물 셋) ·
  `75ef4d3`(FD 승인). 그 위 `22e473c` 가 원 인계 커밋.
- **ADR-068 = A (진행자 결정)** — 데몬이 정책 파일 옆 `<stem>.status.yaml` 에
  `Caps`·`At` 을 쓰고 제어판이 읽는다. 광고·Mediator 무변경.
  `requirements/decisions.md` §2 「탐지 능력 읽기(ADR-068)」 행에 있다.
- **Functional Design 완료·승인됨** —
  `aidlc-docs/nacl1119/construction/panel/functional-design/` 의
  domain-entities · business-logic-model · business-rules.

---

## 2. 남은 단계 (panel · W3 · CP4) — 각 단계 승인 필요

```text
   1  NFR Requirements       심볼 상한 재측정 계획 · 커버리지 80%(internal/panel·internal/proc) ·
                             바인딩/토큰 보안              (승인 지점 3)
   2  NFR Design             위를 설계로                    (지점 4)
   3  Infrastructure Design  건너뛰기 제안 — 로컬 프로세스뿐 · 새 클라우드 자원 없음.
                             진행자 판단                    (지점 5)
   4  Code Generation        Part 1 계획(지점 6) -> Part 2 실행(지점 7)
   5  게이트 CP4             초록이면 PR to main            (지점 8)
```

낼 코드(Code Generation).

```text
   internal/proc           ProcessAlive · SignalStop · OwnsConfig 를 cmd/enodectl/proc_*.go
                           에서 추출.  빌드 태그 짝 · net/http 없음
   internal/panel          New(Config) · Handler().  127.0.0.1:8081
   cmd/enodectl serve <name>   enode 제어판 프로세스를 exec 위임 (setup.go 본뜸)
   cmd/enode panel         os.Args[1]=="panel" 분기로 internal/panel 을 net/http 로 띄움
   internal/enode/status.go(신규)  StatusPath · WriteStatus · ReadStatus +
                           advertise 루프에 guarded 호출 하나
   internal/enode/policy.go   Policy 에 panel_token additive
   임포트 경계 검사 테스트   internal/panel 을 처음 만드는 유닛이 낸다
```

---

## 3. 놓치면 안 되는 설계 결정/제약

- **proc 는 `internal/proc` 로** (net/http 없음). `enode-features.md` 3.1.2 는
  `internal/panel` 이라 했으나 그러면 `cmd/enodectl` 의 stop/start 가 그 패키지를
  거쳐 net/http 에 닿아 **`enodectl.exe` 심볼 상한(CP4)이 깨진다.** 유닛 정본
  `unit-of-work.md` §5 가 이긴다. (business-rules §3)
- **serve 는 exec 위임** — enodectl 이 net/http 에 안 닿게. 심볼 상한
  `enodectl.exe` net/http ≤50 · crypto/tls ≤10 (지금 6·1 · `ci.yml:479` ·
  continue-on-error 없음). 완료 시 재측정.
- **stop 은 cancel 먼저** — `runctl.Client.Cancel(run_id)` 부르고 데몬 끈다.
  Mediator 불통 시 「임대 만료로 죽는다」 경고 후 진행. 재시작도 같은 규칙
  (새 하위명령 아님).
- **완료 조건은 버튼 전부 나열** — status·start·stop·logs + drain 걸기·모드
  (graceful·at-boundary)·풀기. S3(걸기 전)/S3b(건 뒤)/S1/S5. 커버리지 80%.

---

## 4. 접점 (진행자 직렬 병합 주의)

```text
   internal/enode/advertise.go   drain(shin-son · 병합됨)이 만진 파일.
                                 panel 이 status 쓰기 호출 하나 더한다
   internal/enode/policy.go      panel_token additive
   internal/enode/status.go      신규
   cmd/enode/main.go             panel 하위명령 분기
   cmd/enodectl                  serve · proc 추출
   internal/panel                panel·transcript 둘 다 nacl1119 — 한 손
```

**안 건드림** — `internal/store` · `internal/api/ui` · `cmd/mediator` · `design/` ·
루트 상태·감사 파일 · Capabilities 광고 계약(ADR-068=A 라 광고 무변경).

---

## 5. 그다음 — transcript (W4 · CP6) · panel 병합 뒤

- 하네스 출력 링 파일 tee(`runner.go`·`claim.go`) + 제어판 트랜스크립트 카드 +
  지난 작업(`GET /v1/runs` 필터 + record tar). **DB 안 만짐.**
- 링 파일 로직(512KiB · 1초 · WriteAt만 · 감김 · 비우기)은 `decisions.md` §6.3 이
  이미 확정 — 다시 논의 안 한다.
- 브랜치는 그때 `origin/main` 에서 `unit/transcript` 새로 딴다.

---

## 6. 먼저 읽을 정본

`aidlc-docs/nacl1119/construction/panel/functional-design/*` (FD 3종) ·
`requirements/decisions.md` §2(ADR-068 행) · `unit-of-work.md` §5 ·
`scene-gates.md` CP4 · 원 인계문서 `panel-transcript-handoff.md`.
