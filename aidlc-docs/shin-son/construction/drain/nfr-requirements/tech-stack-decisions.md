# drain — 기술 선택

## 1. 고정된 것
```text
   Go 1.26 · pgx/v5 · yaml.v3 (이미 의존 — 설정 파일이 쓴다) · net/http · log/slog · 표준 testing
```
**새 의존 0.** `go.mod` 가 안 움직인다.

## 2. 이 단계가 정한 것 둘

### 2.1 정책 파일은 yaml.v3 로 읽는다
```text
   후보                판정     이유
   yaml.v3            고른다    설정 파일과 같은 형식(decisions §1) · 이미 의존 · 고정 타입 바인딩 (SECURITY-13)
   JSON               버린다    「같은 형식(yaml)」이 정본
   환경변수 · 플래그    버린다    정본은 파일.  제어판이 쓰고 사람이 본다
```

### 2.2 파일 감시 없이 매 광고에 읽는다
```text
   후보                      판정     이유
   광고 직전 읽기 (stat+read)  고른다    「광고 직전에 읽는다」가 정본.  파일 하나 · 주기 5~60초.  의존 0
   fsnotify 류 감시            버린다    새 의존.  윈도우·유닉스 갈래.  얻는 것은 광고 주기 안의 몇 초
   Worker 가 파일을 직접 읽기   버린다    FD Q2 — Worker 의 기준은 중앙이 받아 적은 값(응답)이다
```

## 3. 안 들이는 것
```text
   중앙 drain 라우트 · 정책 테이블      ADR-063 §4 · constraints §7
   파일 잠금 · 원자적 쓰기 도구         읽기만 한다.  쓰는 쪽(panel)의 일
   재시도 큐 · 스케줄러                 답 A — 취소 실패는 로그.  주기에 안 얹는다
```

## 4. 차단 게이트에 미치는 영향
```text
   crypto/tls · net/http 심볼 상한     안 움직인다 — cmd/enodectl 이 이 파일들을 안 딛는다 (obs 실측)
   커버리지 80%                        internal/enode · internal/api · cmd/enode 에 걸린다.  새 문장 전부 덮는다
   스킵 0 · U+2605 0                   그대로
```
