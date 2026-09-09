# transcript — NFR Requirements

이 유닛의 비기능 요구다. 대부분 `decisions.md` §6.3/§6.4 가 이미 못 박았다.
정본 — enode-features 3.1.3 비기능 · `constraints.md` · `decisions.md` §6 ·
`.github/workflows/ci.yml`.

---

## 1. 이식성 · 단일 코드경로 (차단)

```text
   요구                                        수용 기준
   ─────────────────────────────────────────   ────────────────────────────────────────
   링 파일이 윈도우·유닉스에서 같은 코드로 돈다    빌드 태그 쌍이 하나도 안 는다.  Truncate·rename·
   (decisions §6.3)                             삭제·잠금 없이 WriteAt 만 쓴다
   크로스 빌드                                   linux · darwin · windows go build ./... 통과
```

**윈도우가 이 설계를 정했다** — `FILE_SHARE_DELETE` 미제공·Truncate 중 읽기
깨짐 때문에 회전·삭제·자르기를 아예 안 하는 모양을 골랐다. 그래서 OS 분기가 0.

---

## 2. 성능 · 응답성

```text
   링 쓰기      WriteAt 한 겹 · 상한 512 KiB.  단계당 프로세스 stdout 을 tee 하는 비용만.
                하네스 계약(--output-format) 무변경 (decisions §6.2 Q2)
   카드 폴링     로컬 파일 읽기 1초 (화면 열려 있을 때만).  Mediator 폴링(5초)과 별개 타이머.
                네트워크 부하 없음 — 로컬 파일이라 그 이유가 안 걸린다 (decisions §6.2)
   지난 작업     tar 는 지난 것을 눌러 볼 때만 나가고 1초 폴링과 무관 (decisions §6.5)
   심볼 상한     영향 없음 — 새 코드는 internal/enode(encoding/binary·os)와 internal/panel.
                enodectl.exe 링크 그래프에 net/http·crypto/tls 를 안 들인다
```

---

## 3. 신뢰성 · 원자성

```text
   찢긴 읽기     잠금이 없으므로 읽는 쪽이 머리를 다시 읽어 많이 움직였으면 한 번 더 읽는다
   쓰는 중 읽기   된다(FILE_SHARE_READ|WRITE).  읽는 쪽이 커서 밖 오래된 바이트를 무시한다
   비우기 시점    단계 시작 때 Reset — 떠나 있던 사람에게도 방금 끝난 것이 남는다 (decisions §6.2)
   tee 실패 격리  링 쓰기가 막혀도 하네스 실행·Decode·로그는 그대로 (Job.Transcript 는 보조)
```

---

## 4. 시험성 · 커버리지 (차단)

```text
   패키지 커버리지 하한 80%   internal/enode(링 파일·tee) · internal/panel(카드·지난작업 라우트)
   DB 없이 도는 테스트        링 파일은 TempDir · 지난 작업은 httptest 가짜 Mediator + tar 픽스처
   스킵 0                    새 스킵 안 넣는다
```

링 파일은 순수 파일 IO라 단위 테스트가 쉽다 — 쓰고·감기고·비우고·읽어 순서와
상한 초과 동작을 단언한다.

---

## 5. 기술 스택 · 의존 (무변경)

```text
   새 Go 의존   0.  archive/tar · encoding/binary · io 는 표준.  go.mod·packaging 무변경
   Mediator     변경 0.  cmd/mediator · internal/api 안 건드린다 (decisions §6.4)
   하네스 계약   무변경.  stream-json 전환은 이월 (decisions §6.4)
```

---

## 6. 표기 · 언어 (차단)

```text
   glyphscan    비테스트 Go 문자열 리터럴에 장식 문자 0 (카드 UI 는 CSS 로 · 가운뎃점만)
   언어         에러·로그·CLI·테스트는 영어 · 주석·문서는 한국어.  화면 UI 는 시안대로 한국어
```

---

## 7. 이 단계에서 안 정하는 것

```text
   링 형식의 바이트 배치를 코드로     NFR Design·Code Generation (FD 가 형식은 이미 정함)
   카드의 시각 디자인               design/enode-ux.pen S3/S4 를 코드 단계에서 읽어 맞춘다
```
