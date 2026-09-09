# drain — 코드 요약

drain 유닛(W2 · CP3)의 Code Generation Part 2 산출물이다. 코드는 저장소 루트에 있고
여기는 **무엇이 어디에 섰고 무엇으로 쟀는가**다. 계획은
`aidlc-docs/shin-son/construction/plans/drain-code-generation-plan.md`.

---

## 1. 만진 파일

```text
   갈래   파일                               무엇
   ────   ────────────────────────────────   ─────────────────────────────────────────────
   A      internal/contract/advert.go         drain 어휘 상수 셋 (한 벌의 정본)
          internal/store/store.go             상수 셋이 contract 의 별칭이 된다
          internal/store/queue.go             NodeDrain (한 함수)
          internal/store/queue_test.go        시험 하나
   B      internal/enode/policy.go            신규 — PolicyPath · policyReader(읽기 · 접기 · 원인별 한 번 경고 · 권한 경고)
          internal/enode/advertise.go         AdvertResponse.Drain · Advertiser.Held · 광고 직전 readPolicy · 응답 → Held
          internal/enode/leases.go            Held 에 drain · renew 와 겉면 넷 (nil 수신자 안전)
          internal/enode/claim.go             Worker.Run 의 at-boundary 분기 · noteDraining
          cmd/enode/main.go                   Advertiser 에 Held: held 한 줄 (조율 · panel 의 파일)
          internal/enode/policy_test.go       시험 다섯 · drain_test.go 시험 셋 (DB 없음)
   C      internal/api/api.go                 postResult 의 [drain] 갈래 둘(되돌림 · 정산 뒤) · drainAtBoundary
          internal/api/drain_test.go          시험 다섯 (DB)
```

행렬 밖은 `cmd/enode/main.go` 한 줄뿐이다. `internal/enode/claim.go` 는 transcript 가
같은 파일의 다른 자리(runner 의 tee)를 만질 수 있어 조율 항목이다.

## 2. 계획이 정한 것이 코드에서 어떻게 섰나

```text
   정책 파일 → 광고     Advertiser.Run 이 광고 직전에 readPolicy — Ident.Config 옆 <stem>.policy.yaml.
                       없으면 제로값(omitzero 로 키가 안 나간다).  틀리면 "" + 원인이 바뀔 때만 Warn.
                       유닉스에서 그룹·타인 쓰기 비트면 Warn 한 번(느슨해질 때마다).  설정 경로가 없는
                       시험은 안 읽는다
   응답 → Worker        AdvertResponse.Drain → Held.SetDrain (바뀌면 Info "drain acknowledged by mediator") ·
                       Held.SetRenew(광고 주기).  Held 가 nil 이면 안 나른다 — 기존 시험이 오늘 그대로 돈다
   Worker 의 태도       at-boundary 면 claim 을 안 걸고 광고 주기만큼 기다린다.  한 번만 찍는다.  graceful 은 그대로
   Mediator 의 경계     postResult — 되돌림 갈래와 정산 뒤 갈래 둘 다 drainAtBoundary: NodeDrain → at-boundary 면
                       Cancel(run, "drain:<node>") (안에서 큐 깨우기) → 응답 run_state 에 FAILED.
                       마지막 단계는 정산이 먼저 닫는다.  실패는 로그(NFR 답 A)
   nil 안전             기존 시험이 Held 없이 Worker 를 만든다 — 첫 실행에서 SIGSEGV.  Drain()·Renew() 를 nil 수신자에
                       안전하게 고쳤다 (같은 커밋)
```

## 3. 재는 것 — 결과 (2026-09-09T02:23:47Z · PostgreSQL 18.6 · 55434)

```text
   go build · go vet · gofmt · glyphscan(88 파일) · 크로스 빌드 넷      통과
   go test ./internal/enode ./internal/contract ./cmd/enode (DB 없음)   통과
   go test ./internal/store ./internal/api (DB)                        통과 (drain 시험 열넷 포함)
   go test ./... -coverpkg (ci.yml awk)                                 열여섯 패키지 하한 80% 통과 · 전체 87.0%
                                                                       internal/api 80.5% (420/522) · internal/enode 85.0%
   스킵                                                                 0
   흔들린 시험                                                          cmd/enodectl TestCmdStart_ 셋 — 병렬 실행에서 한 번.
                                                                       queue 때와 같은 셋 · 단독 통과 · 이 유닛 밖
```

**CP3 실동작 통과** — enode_cp2 판 · Mediator · 노드 하나(정책 파일 `local.policy.yaml`).

```text
   d-a (단계 둘 · 각 12초) 제출               RUNNING
   파일에 drain: at-boundary                  다음 광고 → GET /v1/nodes draining at-boundary · lease 있음
   단계 1 이 끝난 경계                          FAILED · verdict note "cancelled by: drain:1db1bc59ead5" ·
                                             s1 DONE · s2 FAILED · lease 없음 · record 에 blobs/01.0-o1.txt · steps/01-s1.json
   새 계약 d-b                                202 QUEUED
   파일 지움(해제)                             다음 광고 → d-b RUNNING → SUCCEEDED · draining ""
   graceful                                   d-g 가 끝까지 SUCCEEDED
   로그   mediator  run closed at a step boundary · queued · promoted from queue
          enode     drain acknowledged (at-boundary · "" · graceful) · not claiming · claiming again
```

## 4. 진행자에게
```text
   ①  조율 둘 — cmd/enode/main.go 한 줄 · internal/enode/claim.go 한 분기
   ②  store.go 의 drain 상수 셋이 contract 의 별칭이 됐다 (obs 파일 한 줄)
   ③  internal/api 80.5% — 여유 2 문장.  안 덮인 것은 DB 오류 경로 셋(NodeDrain 실패 · Cancel 실패 · 정산 실패)
   ④  NFR 답 A 의 수락 위험 — 취소 실패 시 소유자의 stop 이 닫는다
```
