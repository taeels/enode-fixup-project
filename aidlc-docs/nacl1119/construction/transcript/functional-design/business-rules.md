# transcript — 규칙 · 검증 · 완료 조건 (business-rules)

정본 — `decisions.md` §6.2/§6.3/§6.4/§6.5 · `unit-of-work.md` §6 · enode-features
3.1.3 · `requirements/scene-gates.md` CP6 · `constraints.md`.

---

## 1. 윈도우가 설계를 정했다 (링 파일)

파일로 이야기하면 윈도우에서 밟을 것이 넷이다 (decisions §6.3).

```text
   쓰는 중 읽기       된다 (os.OpenFile 이 FILE_SHARE_READ|WRITE · 잠금 안 검)
   rename 으로 회전   막힌다 (Go 가 FILE_SHARE_DELETE 를 안 준다) -> 안 한다
   Truncate 하며 읽기 읽는 쪽이 깨진 것을 본다 -> 안 한다
   끝나고 삭제        열려 있으면 못 지운다 -> 안 한다
```

**뒤의 셋을 아예 안 하는 모양이라 윈도우와 유닉스가 같은 코드로 돈다 — 빌드
태그 쌍이 하나도 안 는다.** 크기 고정 · 이름 고정 · 삭제 없음 · 잠금 없음.

---

## 2. 링의 규칙

```text
   용량        512 KiB (decisions §6.2).  80자 줄로 대략 6,400줄
   쓰기        WriteAt 만.  off=total%capacity · 끝에서 감김 · 머리 total 갱신
   상한 초과    안 깨진다.  오래된 바이트부터 덮인다
   비우기       단계 시작 때 total=0·generation+1 (Reset).  끝날 때가 아니다
   찢긴 읽기    잠금이 없으므로 머리를 다시 읽어 많이 움직였으면 한 번 더 (읽기 완화)
   권한        유닉스 0600 · 윈도우 상속 ACL (정책·상태 파일과 같다)
   magic       "ENTR" 아니면 카드가 「아직 없음」 — 다른 파일을 링으로 안 읽는다
```

---

## 3. tee 의 규칙

```text
   흘리는 것    하네스 원문 stdout (decisions §6.2 Q2).  --output-format 을 안 건드린다
   한 겹        runner.go 의 cmd.Stdout 을 io.MultiWriter(&stdout, ring) 로.
               Decode·로그 반환은 그대로 &stdout 을 읽는다 — 하네스 계약 무변경
   명령 단계    claim.go 의 buf 를 io.MultiWriter(&buf, ring) 로
   nil 안전     Job.Transcript 가 nil 이면 안 흘린다 (설정 없는 시험)
   데몬 무지    데몬은 제어판을 모른다 — 파일 하나로만 이야기한다 (새 포트·IPC 0)
```

---

## 4. Mediator 무변경 · 안 넓히는 것 (decisions §6.4)

```text
   cmd/mediator · internal/api 변경 0.  새 라우트 0
   stream-json 전환 · SSE · WebSocket · 트랜스크립트 구독 · Mediator 중계 — 안 한다(이월)
   node= 서버 필터 신설 안 함 — 제어판이 assigned 로 거른다 (§6.5)
   새 Go 의존 0 — archive/tar 는 표준.  go.mod·packaging 무변경
   장면(dhseo) 변경 0 — 트랜스크립트는 그 장면에 안 나오고 CP4 조건이 아니다
```

---

## 5. 완료 조건 — 게이트 CP6 (scene-gates)

두 국면이다. 장면 밖이라 CP4 뒤 아무 때나 닫는다(CP6 이 빨개도 CP4 는 초록).

```text
   도는 것   계약을 던져 놓고 제어판을 열면 단계가 끝나기 전에 카드에 하네스 출력이 흐른다.
            상한을 넘겨도 안 깨지고 오래된 줄부터 밀린다.  다음 단계가 첫 글자를 쓸 때 갈린다(Reset)
   지난 것   이 노드가 한 Run 목록이 뜨고, 하나를 누르면 결과(verdict.checks)와
            봉인된 트랜스크립트(record tar 의 logs/NN-*.log)가 보인다
   둘 다     데몬 로그 카드가 그 옆에 그대로 있고 둘이 다른 물건임이 화면에서 보인다
   화면      S3 안의 카드가 살아난다.  S4 는 자리만 그대로 둔다 (decisions §6.1)
   커버리지   internal/enode 하한 80% 재측정 (링 파일·tee 새 코드 · enode-features 3.1.3 비기능)
```

---

## 6. security-baseline 준수 요약

```text
   규칙          이 유닛에서                                      판정
   ───────────   ──────────────────────────────────────────────   ──────────────
   SECURITY-01   평문 HTTP · 기존 표면과 같다                       기록만 (N/A)
   파일 권한      링 파일 유닉스 0600 · 윈도우 상속 ACL              적용
   SECURITY-08   트랜스크립트는 노드 밖으로 안 나간다 (§6.4)          적용 (경계)
                 지난 것은 제어판이 자기 Mediator 토큰으로 읽는다
```

읽기 전용 카드다 — 하네스 출력을 보이기만 한다. 편집·전송 자리를 안 만든다.

---

## 7. 접점

```text
   internal/enode/runner.go · claim.go   drain(광고 · shin-son) · panel(status 쓰기)과 다른 파일/자리.
                                         tee 한 겹씩 · Job 에 필드 하나
   internal/enode/transcript.go          신규
   internal/panel                        panel·transcript 둘 다 nacl1119 — 한 손 (카드·라우트 추가)
   안 건드림   internal/store · internal/api · cmd/mediator · design/ · 루트 상태 파일
```
