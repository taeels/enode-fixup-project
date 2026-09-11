# 예외사항 — 이 팩이 구현하지 않는 것

요구 문서가 말하지 않은 것을 「빠뜨린 것」이 아니라 「뺀 것」으로 읽게 한다.
끝의 구조 불변식과 접점 절은 유닛과 무관하게 참인 것이다.

## 제외 기능 (구현하지 않음)

### 1. 푸시 표면
- SSE · WebSocket · 롱폴 GET
- Mediator 가 노드를 부르는 어떤 경로

근거 — `ADR-025` §7 ③ 「스트리밍이면 표면의 성격이 바뀐다」. 폴링으로 족하다.
`ADR-014` 결정 2 — Mediator 는 노드에 접속하지 않는다.

### 2. 토큰 단위
- `--include-partial-messages`
- 글자 단위로 자라는 화면

근거 — 메시지 단위면 1초 폴링에 충분하고 링이 덜 감긴다. 앞 팩 `constraints.md`
§5 의 「토큰 단위 중계 제외」는 그대로다.

### 3. 입력 방향
- `--input-format stream-json` · 살아 있는 stdin 으로 대화를 잇는 것
- 화면에서 에이전트에게 말을 거는 것

근거 — 되묻기는 단계를 끊고 `--resume` 으로 잇는다 (`ADR-032` · `HarnessResult.Session`
주석). 살아 있는 파이프는 임대를 사람 답에 묶는다.

### 4. 로그의 가공
- 마스킹 · 검색 · 필터 · 요약
- 파서가 사건을 고치거나 버리는 것

근거 — 원문이 봉인이다 (`ADR-005`). 파서는 읽을 뿐이고 못 읽은 줄은 `raw` 로 남긴다.

### 5. 형식 변경
- 링 파일의 머리 · 크기 · 비우기 · 권한
- Record 의 `logs/NN-<단계>.log` 이름과 자리
- `HarnessResult` 의 필드 (`Reason` 어휘 · `Session` · `Version`)

근거 — 링은 윈도우가 정한 모양이다 (앞 팩 §6.3). 로그 파일의 **내용**이 NDJSON 이
되는 것은 형식이 아니라 하네스가 내는 것이 바뀐 것이다.

### 6. 새 실행파일 · 새 포트 · 새 저장소
- Mediator 밖의 로그 서버 · 별도 스트리밍 프로세스
- DB 스키마 변경. 진행 로그는 디스크의 파일이다

근거 — 앞 팩 3.1.3 「DB 를 안 만진다」 그대로. 파일이 이미 있고 라우트 하나가 그것을 낸다.

### 7. 기존 계약 변경
- `success_when` · 봉인 규칙 · 임대 키 · `I1` · `I2` · `I4` · `I5`
- `GET record` 의 `409`

---

## 보안 확장이 하는 것과 안 하는 것

`security-baseline` 을 켠다. 규칙은 새로 만드는 표면(GET 라우트 · PUT 응답 ·
현황판 카드 · 파서)에만 건다. 기존 코드의 사실은 기록만 한다 — `decisions.md` 3절.

---

## 구조 불변식 — 유닛을 어떻게 가르든 참이다

**여기 있는 것은 제약이지 계획이 아니다.** 어느 유닛이 무엇을 맡는지는 Units
Generation 이 정한다.

### 새 코드가 사는 자리

```text
   internal/transcript   파서.  새 패키지.  표준 라이브러리만
   internal/enode        Argv · Decode · 청크 푸시 · 꼬리 업로드
   internal/api          GET log 핸들러는 새 파일.  api.go 는 등록 줄 하나
   internal/panel        카드가 파서를 쓴다.  지난 것의 출처 전환
   internal/api/ui       Run 상세 카드.  정적 파일
   internal/record       AppendLog 가 총 길이를 돌려준다
```

### 임포트 금지 — 앞 팩의 넷에 둘을 더한다

```text
   internal/panel      ->  internal/store        금지
   internal/panel      ->  internal/api          금지
   internal/api/ui     ->  internal/store        금지
   internal/enode      ->  internal/panel        금지
   internal/transcript ->  internal/enode        금지   파서는 아래를 모른다
   internal/transcript ->  internal/api · store · panel   금지
```

경계 검사 테스트의 표에 두 줄이 는다. 검사기는 표를 읽는다.

### 파서는 하나다

**사건을 그리는 규칙은 파서가 들고 화면은 그것을 그린다.** 제어판의 페이지와
현황판의 정적 파일이 각자 JSON 줄을 해석하기 시작하면 둘이 갈린다. 화면은
`as=events` 나 제어판 API 가 준 사건만 그린다.

### 차단 게이트 다섯 — 그대로 걸린다

```text
   crypto/tls T 심볼 (enodectl.exe)      상한 10
   net/http  T 심볼 (enodectl.exe)       상한 50     panel 은 enode 의 하위명령이라 무관
   패키지별 커버리지                      하한 80%    transcript · enode · api · panel 이 걸린다
   허용목록 밖의 스킵                     상한 0
   U+2605 을 담은 파일 수                 상한 0
```

파서는 하네스 없이 도는 테스트로 채운다 — 실측한 stream-json 줄을 `testdata` 에
둔다. 그것은 실제로 받았던 것의 기록이므로 고치지 않는다 (`CONVENTIONS.md` 2.2).

---

## 접점 — 다른 손과 부딪히는 자리

**팩 둘이 같은 파일을 만진다.** 진행자가 직렬로 병합한다 (`CONVENTIONS.md` 3.1).

```text
   internal/enode/claude.go   Argv       이 팩은 출력 형식을 바꾼다.
                                         harness-components 팩은 MCP 플래그를 더한다
                              Decode     이 팩만
   internal/enode/runner.go   tee · Emit  이 팩.  Job 의 팩 · MCP 필드는 저쪽
   internal/enode/claim.go    UploadLog 호출 자리.  이 팩만
   internal/api/api.go        등록 줄 하나.  다른 유닛도 등록 줄만 만진다
   internal/panel             앞 회차의 nacl1119 가 만든 자리.  이 팩이 카드를 바꾼다
   internal/api/ui            runixs 의 자리.  Run 상세에 카드 하나
```

**Units Generation 에 거는 요구 하나 — 파일 행렬을 반드시 낸다.**
