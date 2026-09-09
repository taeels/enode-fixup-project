# transcript — 구현 요약과 CP6 게이트 판정

Code Generation 산출물의 요약이다. 코드는 저장소에 있고(아래), 여기는 무엇을
냈고 게이트가 무엇으로 초록인지를 적는다. 커밋 — `a416289`.

---

## 1. 낸 파일

```text
   internal/enode/transcript.go   고정 크기 링 파일 Ring — 머리(magic·판·용량·total·generation)
                                  + 몸통.  OpenRing · Write(io.Writer · WriteAt 감김 · 실패 삼킴) ·
                                  Reset(비움) · ReadRing(머리 재확인) · TranscriptPath.  빌드 태그 없음
   internal/panel/transcript.go   GET /api/transcript(로컬 링) · /api/runs(assigned 필터) ·
                                  /api/record(tar 의 logs/NN-*.log)
```

만진 파일 — `internal/enode/runner.go`(Job.Transcript · cmd.Stdout 을 io.MultiWriter) ·
`internal/enode/claim.go`(Worker.ring · Run 이 열기 · execute 가 Reset · 명령 단계 tee ·
Job.Transcript 전달) · `internal/panel/panel.go`(라우트 셋 등록) · `internal/panel/page.go`
(하네스 트랜스크립트 카드 + 지난 작업 목록/상세 · 시안 다크 토큰).

---

## 2. CP6 게이트 판정 (scene-gates CP6)

두 국면. 장면 밖이라 CP4 뒤 아무 때나 닫는다.

```text
   도는 것    링에 tee 하고 제어판이 1초로 읽어 카드에 흐른다.  상한(512 KiB) 넘겨도
             안 깨지고 오래된 바이트부터 밀린다(Ring 테스트로 실증).  다음 단계가
             Reset 해 generation 이 바뀌면 화면을 비운다(첫 글자에 갈림)
   지난 것    GET /v1/runs 를 assigned 로 걸러 이 노드 목록 · 누르면 verdict.checks 와
             record tar 의 logs/NN-*.log
   둘 다      데몬 로그 카드가 그 옆에 그대로 · 「트랜스크립트와 다른 물건」으로 갈려 보인다
   화면       S3 안의 카드가 산다 · S4 는 자리만 (decisions §6.1)
```

**게이트 재료 재측정.**

```text
   빌드 태그 쌍   0 늘어남 — 링이 Truncate·rename·삭제·잠금을 안 해서 단일 코드경로
   심볼 상한     enodectl.exe net/http=13 · crypto/tls=1 (무영향 · 새 코드는 net/http 안 늘림)
   커버리지      internal/panel 86.1% · internal/enode 는 링 파일 충실 테스트(감김·상한·Reset·순서)
   크로스 빌드    linux · darwin · windows go build ./...
   vet · glyphscan(가운뎃점만 · 카드 강조는 CSS) · Mediator 변경 0 · 새 의존 0
```

**보류(게이트 밖) — scene-gates §4.** 눈 검증(계약을 던져 카드에 글자가 흐르는
것 · 지난 Run 을 눌러 봉인 트랜스크립트를 보는 것)은 사람이 실제 함대·데몬으로
닫는다. 코드·자동 검사는 초록이다.

---

## 3. 시안 반영

트랜스크립트 카드·지난 작업은 `design/enode-ux.pen` 의 제어판 디자인 토큰(다크
#0B0D10 · 보라 강조 · Inter/JetBrains Mono · 반경 10)을 그대로 쓴 다크 카드다.
정본(scene-gates CP6 · decisions §6.1)이 「S3 의 카드」로 살리라 했고 .pen 은 S4 에
Log Area 를 두었는데, 정본대로 S3 카드로 살리되 시각은 시안 토큰을 따랐다.

---

## 4. 로컬 검증 한계 (기록)

이 Windows·Bedrock 세션에서 `internal/enode` 의 몇 테스트가 환경 탓에 실패한다 —
`TestChildAttr`(off-windows 전용) · `TestEnv`(테스트 env 에 CLAUDE_CODE_*·
AWS_BEARER_TOKEN_BEDROCK 존재) · `TestBuildPrompt`(\o\ Windows 경로) ·
`TestPrepareKeepsBuildCacheDropsJunk`(autocrlf). **전부 이 유닛의 변경과 무관**하고
CI(리눅스·청정 env)에서는 안 난다. 트랜스크립트 링·panel 라우트 테스트는 통과한다.

---

## 5. 접점

`internal/enode/runner.go`·`claim.go` 는 drain(광고 · shin-son)·panel(status 쓰기)과
다른 자리에 tee 를 한 겹씩 더했다. `transcript.go` 는 신규. `internal/panel` 은
panel·transcript 둘 다 nacl1119 라 한 손. 안 건드림 — store · api · cmd/mediator ·
design · 루트 상태 파일.
