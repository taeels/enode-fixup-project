# transcript — Code Generation 계획 (Part 1)

FD·NFR 이 값을 다 정했으므로 기계적이다 — 결정 없이 Part 2(구현)로 이어간다.
정본은 이 유닛의 functional-design/ · nfr-design/ · `decisions.md` §6.

---

## 낼 것 · 순서 (의존 순)

- [ ] 1. `internal/enode/transcript.go` 신규 — `Ring`(머리 32B: magic·version·capacity·
      total·generation + 몸통) · `TranscriptPath` · `OpenRing` · `(*Ring) Write`(io.Writer ·
      WriteAt 감김 · 링 쓰기 실패는 삼킨다) · `Reset` · `Close` · `ReadRing`(재확인). 빌드 태그 없음
- [ ] 2. `internal/enode/runner.go` — `Job` 에 `Transcript io.Writer` additive. `cmd.Stdout` 을
      `io.MultiWriter(&stdout, j.Transcript)`(nil 아니면). Decode·로그 반환 그대로
- [ ] 3. `internal/enode/claim.go` — 명령 단계 `buf` 를 `io.MultiWriter(&buf, ring)` 로 (ring 주입)
- [ ] 4. Worker 배선 — 노드의 Ring 을 열고(`OpenRing(TranscriptPath(Ident.Config),512*1024)`),
      CLAIMED 시작 때 `Reset()`, 하네스에 `Job.Transcript=ring`, 명령 단계에 같은 ring
- [ ] 5. `internal/panel` — `GET /api/transcript`(ReadRing -> {generation,total,data}) ·
      `GET /api/runs`(runctl.Runs -> assigned 필터 목록) · `GET /api/record?run=`(runctl.Record
      tar -> logs/NN-*.log + verdict.checks)
- [ ] 6. `internal/panel/page.go` — 트랜스크립트 카드(1초 폴링 · generation 바뀌면 비움) +
      지난 작업 목록/상세. **design/enode-ux.pen S3/S4 를 pencil MCP 로 읽어 다크 토큰에 맞춘다**
- [ ] 7. 테스트 — 링(감김·상한 초과·Reset·순서 · TempDir) · tee · /api/transcript · /api/runs·
      /api/record(httptest + tar 픽스처). 커버리지 80%
- [ ] 8. 검증 — 크로스 빌드 3종 · go vet · glyphscan · gofmt · 커버리지 80% · 심볼 상한 재측정(무영향 확인)

---

## 완료 조건 (게이트 CP6)

business-rules §5 — 도는 것(단계 끝 전 흐름 · 상한 넘겨도 앞부터 밀림 · 다음 단계
첫 글자에 갈림) · 지난 것(verdict.checks + 봉인 트랜스크립트) · 데몬 로그 카드와
다른 물건임이 화면에 · S4 자리만 · 커버리지 80%.

---

## 안 하는 것 (decisions §6.4)

stream-json · SSE/WebSocket · Mediator 변경 · node= 서버 필터 · 새 의존 · 장면 변경.
