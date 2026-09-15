# 유닛 — 정의와 책임

**여덟이다** (Q1 = A). 가르는 기준은 **되돌리기 단위와 게이트 조각**이다.

```text
   담당      한 손 (taeels).  질문 3 = B
   브랜치    unit/<유닛>.  v3-run-transcript 에서 딴다.  PR 로 main
   문서 루트  aidlc-docs/taeels/ (Construction).  회차 루트가 아니다
   게이트    Q2 = A — **그 게이트가 재는 기능을 마지막으로 완성하는 유닛이 진다.**
             앞선 유닛은 코드 게이트(빌드 · 시험 · 커버리지 80% · 경계 검사)로 병합한다
```

**「딛는 게이트」 칸이 Q2 = A 의 대가다.** 앞선 유닛이 `main` 에 들어간 뒤
게이트가 빨개질 수 있다. 그때 **어느 유닛으로 돌아가야 하는지**를 이 칸이 적는다.

---

## U1 `transcript` — 파서 패키지

```text
   한다       새 패키지 internal/transcript.  사건을 읽는 Parse 와
              껍데기를 짓는 셋(ParseLine · Shell · ElidedMarker).
              셋은 internal/enode/runner.go 에서 옮겨 온다 (D5 · Q4 = A)
              selectLogs 가 그것을 임포트하게 고친다
              경계 검사 표에 임포트 금지 두 줄을 더한다
   안 한다     화면 · HTTP · 파일 · 시계.  사건을 고치거나 버리는 것
   경로       internal/transcript (신규) · internal/enode/runner.go ·
              internal/panel/boundary_test.go
   FR         FR-3
   지는 게이트  없다.  코드 게이트만 — 커버리지 80% 를 **하네스 없이** 채운다
   딛는 게이트  없다.  가장 먼저 선다
   NFR 요구    **돈다.**  파서가 새 표면이고 신뢰할 수 없는 입력을 받는다 (SECURITY-13)
```

## U2 `node-stream` — 사건을 흘린다

```text
   한다       Decode 가 줄 단위로 읽어 emit 한다 (시그니처는 안 바뀐다).
              runner.go:186 의 링 tee 를 다시 잇는다 — 선별을 링에는 안 건다
   안 한다     Argv.  짝 팩이 이미 지났다.  봉투 파서(ParseClaude · lastJSONObject)
   경로       internal/enode/claude.go · runner.go · claim.go
   FR         FR-1 · FR-2
   지는 게이트  없다.  CB1 의 재료다
   딛는 게이트  없다.  U1 을 안 기다린다 — 링 tee 도 Decode 도 파서를 안 쓴다
   NFR 요구    **스킵.**  새 외부 표면 0.  링의 형식 · 크기 · 권한(0600)은 앞 팩이
              값으로 닫았고 잔여 ③ 은 requirements.md 5.4 에 이미 적혀 있다
```

## U3 `progress-store` — 진행 파일의 주인

```text
   한다       AppendProgress · ReadProgress · DropProgress.
              자리는 <Root>/progress/run-<safe(id)>/NN-<safe(name)>.log (D1 · Q2 = A)
              시도가 바뀌면 비운다 (D7 · Q1 = A).  총 길이로 상한을 건다
              Seal 이 tar 보다 먼저 진행 트리를 지운다
              AppendLog 의 반환값 뜻을 총 길이로 바꾼다 (D4)
   안 한다     HTTP.  라우트.  파서
   경로       internal/record/record.go (+ 새 파일)
   FR         FR-5 의 Mediator 쪽
   지는 게이트  없다.  CB3 의 재료다
   딛는 게이트  없다.  U1 을 안 기다린다 — 바이트를 다루지 사건을 안 다룬다
   NFR 요구    **돈다.**  **N2 (진행 파일이 디스크에 앉아 있는 기간의 상한)를 진다**
```

## U4 `log-api` — 라우트 하나와 갈림 하나

```text
   한다       GET /v1/runs/{run}/steps/{seq}/log 신규 (새 파일).
              api.go 에 등록 줄 하나.  PUT 에 ?progress=1&attempt=<n> 갈림
              응답 헤더 넷 — Bytes · Source · Attempt · Capped
              Sealed(runID) 로 봉인 전후를 가른다 (D3)
              as=events 면 U1 의 파서를 거친다
              입력 검증 — seq · from · as · name · attempt (SECURITY-05)
   안 한다     판정을 싣는 것 (ADR-065).  GET record 의 409 를 건드리는 것
   경로       internal/api/log.go (신규) · internal/api/api.go
   FR         FR-6 · FR-5 의 api 쪽
   지는 게이트  **CB0** — grep -c 'mux.HandleFunc' internal/api/api.go 가 17 -> 18.
              **CB0 은 유닛마다 도는 기계 게이트이기도 하다** (기동이 안 깨졌다)
   딛는 게이트  없다.  U1 · U3 을 코드로 딛는다
   NFR 요구    **돈다.**  **N1 (함대 규모에서의 청크 PUT 과 폴링 부하)을 진다**
```

## U5 `panel-live` — 제어판의 도는 것

```text
   한다       handleTranscript 가 링을 읽어 **사건 배열로** 낸다 (원문 토글용 data 도)
              카드가 문장 · 도구 이름 · 접힌 결과를 그린다.  원문 토글
              NC-1 경과 · NC-2 링 잘림 · NC-3 링 잔여 한 줄
              page.go:15 에 보안 헤더 다섯 (requirements.md 5.5)
   안 한다     지난 것.  U6 이 한다.  Mediator 를 안 탄다 — 링을 직접 읽는다
   경로       internal/panel/transcript.go · page.go
   FR         FR-4 의 절반
   지는 게이트  **CB1** — 이 팩의 이유다.  사람이 실제 하네스로 봐야 초록이다
   딛는 게이트  없다.  U1 · U2 를 코드로 딛는다.  **CB1 이 빨가면 그 셋 중 하나다**
   NFR 요구    **돈다.**  카드가 새 표면이고 신뢰할 수 없는 출력을 그린다
```

## U6 `panel-past` — 제어판의 지난 것

```text
   한다       handleRecord 의 출처를 GET 라우트로 바꾼다.  Status 로 단계 목록을 받고
              단계마다 StepLog 을 부른다.  **archive/tar 임포트가 사라진다**
              봉인 뒤의 「본문 N 개가 걷혔다」를 그린다 (NC 없이 CB2 가 잰다)
              internal/runctl 에 클라이언트 메서드 StepLog 하나
   안 한다     verdict.  지금처럼 Run 목록에서 온다
   경로       internal/panel/transcript.go · internal/runctl/client.go
   FR         FR-4 의 나머지
   지는 게이트  **CB2**
   딛는 게이트  **CB1** (U5).  코드로는 U4 를 딛는다
   NFR 요구    **스킵.**  출처만 바뀐다.  새 외부 표면 0 — 클라이언트 메서드가
              기존 do() 를 그대로 탄다
```

## U7 `chunk-push` — 노드가 올린다

```text
   한다       업로더.  tee 의 갈래 하나로 꽂힌다 — io.MultiWriter(ring, uploader)
              주기 2초 · 64 KiB 면 즉시.  응답의 총 길이로 오프셋을 맞춘다
              **실패가 실행도 링도 안 막는다.**  상한에 닿으면 더 안 올리고 노드 로그에
              Client.PutLogChunk 하나
   안 한다     단계 끝의 선별본 업로드.  claim.go:796 이 오늘 그대로 한다
   경로       internal/enode/claim.go · internal/enode/upload.go (신규)
   FR         FR-5 의 노드 쪽
   지는 게이트  **CB3**
   딛는 게이트  **CB1** (U5).  코드로는 U2 · U4 를 딛는다
   NFR 요구    **돈다.**  청크 푸시가 새 표면이다
```

## U8 `fleet-card` — 현황판의 단계 카드

```text
   한다       Run 상세에 단계마다 트랜스크립트 카드.  2초 타이머 — 목록의 5초와 별개
              as=events 로 받아 제어판과 같은 모양으로 그린다.  원문 토글은 as=raw
              NC-6 마지막 갱신 시각.  그래프와 목록을 안 가린다
              화면은 이 회차의 새 .pen — **진행자가 그린다** (CLAUDE.md 의 layering)
   안 한다     Go 코드.  internal/api/ui/ui.go 의 diff 가 0 이다
   경로       internal/api/ui/static/shared/fleet/*.mjs · fleet.css ·
              internal/api/ui/tests/*.mjs
   FR         FR-7
   지는 게이트  **CB4** · **CB6** (한 장면 — 마지막 유닛이다)
   딛는 게이트  **CB3** (U7).  코드로는 U4 를 딛는다
   NFR 요구    **돈다.**  카드가 새 표면이다
```

---

## 9. 유닛과 게이트의 표

| 유닛 | FR | 지는 게이트 | 딛는 게이트 | NFR 요구 | N 값 |
|---|---|---|---|---|---|
| U1 `transcript` | FR-3 | (코드만) | — | 돈다 | — |
| U2 `node-stream` | FR-1 · FR-2 | (코드만) | — | 스킵 | — |
| U3 `progress-store` | FR-5 (med) | (코드만) | — | 돈다 | **N2** |
| U4 `log-api` | FR-6 · FR-5 (api) | **CB0** | — | 돈다 | **N1** |
| U5 `panel-live` | FR-4 (절반) | **CB1** | — | 돈다 | — |
| U6 `panel-past` | FR-4 (나머지) | **CB2** | CB1 | 스킵 | — |
| U7 `chunk-push` | FR-5 (노드) | **CB3** | CB1 | 돈다 | — |
| U8 `fleet-card` | FR-7 | **CB4** · **CB6** | CB3 | 돈다 | — |

**게이트 일곱에 지는 유닛이 다 있다.** CB5 는 「해당 없음」이다 — `runctl mcp` 가
`main` 에 없다 (`requirements.md` 2.5). 유닛이 0 인 것이 빠뜨린 것이 아니다.

**FR-8 도 유닛이 0 이다** — 이월이다. 같은 이유다.

## 10. NFR Requirements 를 도는 유닛 여섯 · 스킵 둘

`execution-plan.md` 3절이 「새 표면을 만드는 유닛만 돈다」로 걸었다.

```text
   돈다    U1 파서 · U3 진행 파일 · U4 라우트 · U5 카드 · U7 청크 · U8 카드
   스킵    U2 — 새 외부 표면 0.  링의 값은 앞 팩이 닫았다
          U6 — 출처만 바뀐다.  기존 do() 를 탄다
```

**스킵하는 둘도 그 자리에 근거를 적는다.** 안 적으면 다음 사람이 빠뜨린 것으로 읽는다.

## 11. 이 문서가 안 하는 것

```text
   유닛의 내부 설계     Functional Design.  유닛마다 돈다
   파일 행렬            unit-of-work-file-matrix.md 가 낸다
   착수 순서            unit-of-work-dependency.md 가 낸다
   스토리 사상          unit-of-work-story-map.md 가 낸다
```
