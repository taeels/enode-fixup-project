# drain — NFR Requirements 계획

AI-DLC Construction · **drain 유닛**(W2 · CP3)의 NFR Requirements Part 1 이다. 답이 오면
`aidlc-docs/shin-son/construction/drain/nfr-requirements/` 아래 산출물 둘을 낸다.

정본 — 팩 `requirements/`(decisions §1·§2·§3 · constraints) · 확장 `security-baseline.md` ·
이 유닛의 Functional Design 셋 · queue 와 obs 의 NFR 표(잇는다).

---

## 0. 이 단계를 도는 이유 · 짧은 이유

보안 확장이 켜져 있어 규칙 열다섯의 판정을 유닛마다 낸다. 이 유닛은 **새 HTTP 표면 0 ·
새 파일 표면 하나(정책 파일 · 읽기만)**라 물을 것이 적다 — FD 의 Q3 가 권한을 이미 닫았다.

## 1. 착수 전에 실측한 것

```text
   광고 주기        기본 60초 · Mediator 의 renew_seconds 가 이긴다 (ADR-028).  CP2 실측은 5초로 돌렸다
   정책 파일 읽기    광고마다 파일 하나 stat+read.  yaml.v3 는 이미 의존이다 (go.mod)
   Worker 고루틴    claim 은 롱폴(최대 2시간).  at-boundary 대기는 그 자리에 select 로 든다
   postResult       ReportStep · SettleIfDone 이 각자 커밋한다.  취소는 셋째 tx
   internal/api     커버리지 80.6% · 여유 3 문장 (queue 가 남긴 것).  postResult 갈래가 문장을 더한다
   internal/enode   84.7%.  policy.go 는 DB 없이 전부 덮인다
   cmd/enode        97.2%.  한 줄 추가
```

## 2. 이미 닫힌 값 · 이 계획이 채우는 권장값

```text
   닫힌 값     지연 상한 없음 · 자동 복귀 없음 · 기본 graceful · 광고 계속 · 중앙 라우트 0 · 정책 파일 권한(FD Q3)
   권장값 (물음으로 안 올린다)
     Worker 의 at-boundary 대기 길이   광고 주기(Held.Renew).  없으면 5초.  근거 — 해제는 광고 응답으로만 알 수 있다
     정책 오류 로그의 빈도             원인이 바뀔 때만 Warn.  근거 — 5초마다 같은 줄이 쌓이면 로그가 사실보다 커진다
     정책 읽기의 캐시                  없음.  파일 하나 · 광고 주기.  근거 — 「광고 직전에 읽는다」가 정본
     취소 로그                         Info 한 줄 — run · node.  계약 본문 없음 (SECURITY-03)
     가용성 수치                       없음 (queue · obs 와 같은 근거)
```

## 3. 물음 하나

### 물음 1
at-boundary 취소(`postResult` 의 셋째 tx)가 DB 오류로 실패하면 그 Run 은 이 노드의 다음
단계에서 멈춘다 — Worker 가 안 집고, 임대는 하트비트가 계속 갱신하므로 회수도 안 온다.
소유자의 `stop`(panel · CP4)이나 다른 노드의 결과 보고가 닫을 때까지다.

A) 재시도 없음. Error 로그 한 줄. 드문 DB 오류이고 `stop` 이 있다 — 새 코드 0 (권장)

B) 하트비트에서 재시도 — `postNodes` 가 「이 노드가 at-boundary 이고 임대를 쥔 Run 이 있고
   이 노드에 CLAIMED 단계가 없다」를 보면 그 Run 을 취소한다. 광고 주기 안에 닫히나 postNodes 에
   질의와 갈래가 는다(api 접점 · 커버리지 여유 3 문장)

C) Other (please describe after [Answer]: A tag below)

[Answer]:

## 4. 실행 계획

- [x] 4.1 물음의 답을 받는다
- [x] 4.2 SECURITY 열다섯을 drain 의 산출물에 대고 판정한다 (FD rules §5 를 넓힌다)
- [x] 4.3 비-보안 NFR 다섯 축 — 성능 · 확장 · 가용 · 신뢰 · 유지보수
- [x] 4.4 기술 스택 — 새 의존 0 · yaml.v3 재사용 · 파일 감시(fsnotify 류) 안 들이는 이유
- [x] 4.5 산출물 둘 · 정합 검사 · 상태 · 감사

## 5. 낼 파일

```text
   aidlc-docs/shin-son/construction/drain/nfr-requirements/nfr-requirements.md
   aidlc-docs/shin-son/construction/drain/nfr-requirements/tech-stack-decisions.md
```
