# 유닛 의존 · 파일 행렬 — Units Generation

AI-DLC Units Generation 산출물이다. 여덟 유닛의 **의존 그래프**와 팩이 필수로
요구한 **파일 행렬**(constraints 구조 불변식)을 진다. 유닛 정의는
`unit-of-work.md`, 게이트·기능 매핑은 `unit-of-work-story-map.md`.

병합 순서와 병렬 착수가 이 그래프에 걸린다 (CONVENTIONS 3.1).

---

## 1. 의존 행렬 (누가 누구를 딛나)

행이 열을 딛는다. `O` 는 의존, `완료` 는 착수는 먼저 되나 완료(게이트)가 그
대상을 기다린다는 뜻이다.

```text
   딛는 쪽 \ 대상   obs   queue  drain  ui   panel  transcript  mcp  demo-back
   ─────────────   ───   ─────  ─────  ───  ─────  ──────────  ───  ─────────
   obs                                                                       
   queue            O                                                        
   drain                  O                                                  
   ui               O                                                  완료  
   panel            O            완료                                        
   transcript       O                        O                              
   mcp              O                                                        
   demo-back        O     O                                                  
```

의존 유형:

```text
   queue -> obs           CP2 가 GET /v1/runs 로 QUEUED 를 본다 · store 접점
   drain -> queue         at-boundary·해제가 WakeQueued 를 부른다
   ui -> obs              브라우저가 GET /v1/nodes·runs 를 부른다 (착수)
   ui -> demo-back        데모 모드의 새 작업이 데모 제출 라우트를 부른다 (완료)
   panel -> obs           현재 작업을 runctl.Client.Nodes 로 조회 (착수)
   panel -> drain         제어판 drain 토글이 쓰는 정책 파일 형식 (완료 · CP4)
   transcript -> panel    트랜스크립트 카드가 사는 화면
   transcript -> obs      지난 작업이 GET /v1/runs 를 읽는다
   mcp -> obs             도구가 runctl.Client.Nodes·Runs 와 그 라우트를 감싼다
   demo-back -> obs       submitter 컬럼
   demo-back -> queue     submit -> 202/QUEUED 경로
```

**순환 없음.** internal 끼리는 임포트 금지 넷이 순환을 막는다 (아래 5절).

---

## 2. 병렬 웨이브 · 임계 경로

```text
   W0   obs                              토대.  모두가 딛는다
   W1   queue · mcp · ui(실 함대 모드)     obs 만 딛는다 — 병렬
   W2   drain(queue) · demo-back(obs·queue)
   W3   panel(obs·drain 완료) · ui(데모 모드 = demo-back 완료)
   W4   transcript(panel·obs)

   임계 경로   obs -> queue -> drain -> panel -> transcript  (다섯 깊이)
   가치 게이트  CP10 = ui(데모 모드) · demo-back · 하드웨어.  W3 에 닫힌다 (맨 끝 아님)
   CP7        ui(표시) · mcp(도구)가 W1 에 서면 CP1 뒤 언제든 잰다
```

**의존 없는 유닛부터 병렬 착수한다.** W1 셋(queue·mcp·ui)은 obs 병합 뒤 동시에
딴다. 착수와 완료가 다른 유닛(ui·panel)은 착수는 이르게, 완료(게이트)는 대상
병합 뒤에.

`ui` 가 그 대표다 — 실 함대 모드(S0~S2 · CP1·CP2)는 obs 만 딛어 **W1 에 착수·완료**,
데모 모드(새 작업 제출 · CP8·CP9·CP11)는 demo-back 의 제출 라우트를 딛어 **완료가 W3**.
`ui` 와 `demo-back` 은 제출 라우트 계약(경로·payload)을 정하면 병렬로 짜고, CP9
end-to-end 검증만 demo-back 병합 뒤다. `panel` 도 같다 — obs 로 착수, drain 으로 완료(CP4).

**담당 배정과 handle 은 `aidlc-docs/construction-roster.md`** 가 진다 — W0 obs(taeels) ·
W1 queue(shin-son)·mcp(taeels)·ui 실모드(runixs) · W2 drain(shin-son)·demo-back(runixs) ·
W3 panel(shin-son)·ui 데모모드(runixs) · W4 transcript(shin-son) · card-news(nacl1119 ·
construction 완료·계속 업데이트).

---

## 3. 파일 행렬 (팩 필수 · constraints 구조 불변식)

유닛마다 만지는 파일을 센다. `N` 신규 · `W` 쓰기(기존 파일 수정) · 빈칸 안 만짐.
**둘 이상이 W 인 파일이 접점**이고 처리를 오른쪽에 적는다.

```text
   파일                            obs queue drain ui panel trans mcp demo   처리
   ─────────────────────────────   ─── ───── ───── ── ───── ───── ─── ────   ──────────────
   internal/store/store.go          W    W               W     store 접점 · 진행자 직렬
   internal/api/api.go              W    W                          W   api.go 접점 · 진행자 직렬
   internal/api/nodes.go            N                                       단독 obs
   internal/api/runs.go             N                                       단독 obs
   internal/api/demo.go                                              N   단독 demo-back
   internal/runctl/client.go        W                                       단독 obs
   internal/enode (광고·정책)              W                              단독 drain
   internal/enode/runner.go·claim.go            W          조율(다른 파일 · 같은 패키지)
   internal/proc/* (신규)                          N                       단독 panel
   internal/panel/* (신규)                         N     W          panel 접점 · 진행자 직렬
   cmd/enodectl                                    W                       단독 panel
   cmd/enode                                       W                       단독 panel
   internal/mcp/* (신규)                                       N           단독 mcp
   cmd/runctl/main.go                                          W           단독 mcp
   internal/api/ui/* (신규)                    N                          단독 ui (Q2=C)
   cmd/mediator/main.go                  W       W                        mediator 접점 · 진행자 직렬
   경계 검사 테스트 (신규)                          N                       단독 panel
   internal/match/match.go          무변경 (호출자 api submit 이 busy 합침)
```

---

## 4. 접점 넷 — 진행자 직렬 병합

```text
   internal/store/store.go     obs(읽기경로) · queue(QUEUED·wake·draining) ·
                               demo-back(submitter 쓰기).  additive 라 충돌 작다(Q3=A)
   internal/api/api.go         obs(nodes·runs 등록 + getRun) · queue(submit 202) ·
                               demo-back(demo 등록).  새 핸들러는 새 파일 · api.go 는 등록 줄만
   internal/panel              panel(생성) · transcript(카드).  panel 이 먼저(생성),
                               transcript 가 카드를 더한다
   cmd/mediator/main.go        queue(기동 wake) · ui(/ui/ 마운트)
```

**진행자가 한 번에 하나씩 병합한다** (CONVENTIONS 3.1·3.2). `internal/enode` 는
drain·transcript 가 다른 파일을 만져 하드 접점은 아니나 같은 패키지 커버리지를
공유하므로 조율한다. 상태 파일 둘(`audit.md`·`aidlc-state.md`)과 `design/` 은
유닛 브랜치가 안 건드린다 — 진행자·shonsin 영역.

**Q2=C 효과** — `internal/api/ui` 가 ui 단독 소유가 되어 접점이 다섯에서 넷으로 준다.

---

## 5. 임포트 금지 넷 · 경계 검사 · 심볼 상한

```text
   internal/panel   -> internal/store    금지
   internal/panel   -> internal/api      금지
   internal/api/ui  -> internal/store    금지
   internal/enode   -> internal/panel    금지 (cmd/enode -> panel 은 허용)
```

**경계 검사 테스트는 panel 유닛이 낸다** — `internal/panel` 을 처음 만드는 유닛이라
그렇다. CP0 의 조건이 아니다(그때는 검사할 패키지가 없어 항진명제).

**심볼 상한(차단 게이트)** — `enodectl.exe` 의 `net/http` T 심볼 ≤50 · `crypto/tls` ≤10.
panel 유닛이 진다 — `enodectl serve` 가 HTTP 서버를 직접 안 세우고 exec 위임하고,
`internal/proc` 가 `net/http` 를 안 물어 상한이 안 깨진다. 유닛 완료 조건에 재측정.

허용이 걸리지 않는 자리 — `panel/mcp -> runctl` · `enodectl/panel -> proc`.

---

## 6. 병합 순서

```text
   1  obs                        (W0 · 토대 병합)
   2  queue · mcp · ui           (W1 · 병렬 · 게이트 초록 뒤 각자 병합)
   3  drain · demo-back          (W2)
   4  panel                      (W3 · obs·drain 뒤)
   5  transcript                 (W4 · panel 뒤)
   ui 데모 모드 완료              demo-back 병합 뒤 (W3)
```

각 유닛은 `unit/<유닛>` 위에서 돌고 **그 유닛의 장면 게이트가 초록이 된 뒤**
`v1-run-dhseo` 로 병합한다. 게이트가 유닛 밖 이유로 빨가면 `scene-gates.md` §4 의
보류로 audit 에 적고 넘어간다 — 보류는 병합 지점이 아니다. 접점 파일은 진행자가
직렬로 병합한다.
