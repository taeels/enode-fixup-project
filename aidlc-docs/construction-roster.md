# Construction 배정 — handle 과 회차별 배정

**1절의 handle 표가 살아 있는 값이다.** `CLAUDE.md` 가 문서 루트
`aidlc-docs/<handle>/` 를 여기서 읽는다.

**2 ~ 6절은 `v1-run-dhseo` 회차(대회)의 배정 기록이고 그 회차는 끝났다.** 배정은
회차마다 다르므로 여기 이월하지 않는다 — 새 회차의 배정은 그 회차 문서 루트의
Units Generation 산출물이 낸다.

```text
   유닛 정본     aidlc-docs/<회차>/inception/application-design/unit-of-work.md
   의존·행렬     같은 자리의 unit-of-work-dependency.md
   게이트 매핑   같은 자리의 unit-of-work-story-map.md
```

## 1. 담당 (handle · 이름)

```text
   shin-son   손신
   nacl1119   문태호
   taeels     최태양
   runixs     김태완
```

## 2. 웨이브 x 담당 x 유닛 (v1-run-dhseo · 기록)

```text
   웨이브  유닛               담당       게이트         의존
   ─────   ────────────────   ────────   ────────────   ──────────────────
   W0     obs                taeels     CP1            (토대 · 의존 없음)
   W1     queue              shin-son   CP2            obs
          mcp                taeels     CP5 (+CP7 도구) obs
          ui (실 함대 모드)    runixs     CP1·CP2        obs
   W2     drain              shin-son   CP3            queue
          demo-back          runixs     CP9            obs · queue
   W3     panel              nacl1119   CP4            obs · drain
          ui (데모 모드)       runixs     CP8·CP9·CP11   demo-back(제출 라우트)
   W4     transcript         nacl1119   CP6            panel · obs

   별개   card-news          nacl1119   CP8            construction 완료 · 계속 업데이트
```

## 3. ui 는 두 층이다 (착수 W1 · 완료 W3)

runixs 의 `ui` 는 한 유닛이지만 딛는 게 갈린다.

```text
   실 함대 모드 (CP1·CP2)      obs 만 딛는다        -> W1 착수·완료
   데모 모드 (CP8·CP9·CP11)    demo-back 제출 라우트 -> W3 완료
```

`ui` 와 `demo-back` 은 **제출 라우트 계약**(경로·payload)을 정하면 병렬로 짠다.
착수는 서로 안 막고, **CP9 end-to-end 검증만 demo-back 병합 뒤**다.

## 4. 박스 그래프

```text
   W0          W1                    W2               W3                  W4
   ───         ─────────────         ──────────       ─────────────       ──────────

   obs   ->    queue          ->     drain      ->    panel        ->     transcript
   taeels      shin-son              shin-son         nacl1119            nacl1119
   CP1         CP2                   CP3              CP4                 CP6

         ->    mcp
               taeels · CP5

         ->    ui (실 모드)   ............................->   ui (데모 모드)
               runixs                            |             runixs
               CP1·CP2                           |             CP8·CP9·CP11
                                     demo-back --+ (제출 라우트)
                                     runixs · CP9

   별개  card-news -- nacl1119 (construction 완료 · 계속 업데이트) · CP8
```

화살표는 의존이다. 세로 등뼈 `obs -> queue -> drain -> panel -> transcript` 가
임계 경로(다섯 깊이). 등뼈를 **shin-son 이 queue·drain, nacl1119 가 panel·transcript**
로 나눠 진다 — 넘겨받는 자리(drain -> panel)가 shin-son -> nacl1119 다. 점선은 `ui` 가
W1 에 착수해 W3 에 완료로 걸치는 것 — demo-back(W2)의 제출 라우트를 딛는다. 가치
게이트 **CP10**(데모 완주)은 ui(데모 모드)·demo-back·하드웨어가 W3 에 닫는다.

## 5. 접점 (v1-run-dhseo · 기록)

**문서 루트 · 브랜치 · 병합의 정본은 `CONVENTIONS.md` 3절과 `CLAUDE.md` 의 문서 루트
규약이다.** 여기 두 벌로 두지 않는다 — 대회 때 두 벌이었고 그래서 갈렸다
(`CONVENTIONS.md` 3.1 은 유닛이 회차 브랜치로 모인다고 적었는데 실제로는 PR 이
`main` 으로 갔다). 그 회차에서만 참인 접점만 남긴다.

```text
   여러 담당이 만진 파일   internal/store · internal/api/api.go · internal/panel ·
                        cmd/mediator/main.go.  PR 을 직렬로 병합하고
                        api.go 는 등록 줄만 (constraints)
   한 손 안이었던 것      internal/panel 은 panel·transcript 둘 다 nacl1119
   나눠 가진 것          internal/api/ui 를 runixs(ui)와 nacl1119(card-news)가
```

## 6. 열린 미정 (v1-run-dhseo · 기록)

```text
   Capabilities{Caps,At} 읽기 계약 (ADR-068)   panel · nacl1119.  진행자 decisions
   sandbox 표시 출처 (decisions §8.5)           ui · runixs.  있는 값을 읽는다 · FD
```
