# U3 `progress-store` — 기술 선택

`nfr-requirements.md` 가 **무엇을 지켜야 하는가**이고 여기는 **무엇으로 짓는가**다.
계획과 답 넷은
`aidlc-docs/taeels/construction/plans/progress-store-nfr-requirements-plan.md`.

---

## 1. 한 줄 — **고른 것이 0 이다**

```text
   새 의존       0
   새 저장소     0
   새 실행파일    0
   새 포트       0
   새 설정 키    0
   새 타이머     0
   새 빌드 태그   0
```

**이 유닛의 기술 결정은 「안 들인다」 일곱이다.** 아래가 그 근거이고, 마지막
2.6 이 **들일 뻔했던 하나**를 적는다 — 그것이 이 단계의 유일한 실제 갈림이었다.

---

## 2. 안 들인 것 일곱

### 2.1 새 의존 0 — 표준 라이브러리만

```text
   쓰는 것   os · io · sync · time · path/filepath · fmt · strconv · encoding/json
            **전부 오늘 internal/record 가 이미 쓰거나 표준 라이브러리다**

   go.mod    diff 0.  go.sum   diff 0
   툴체인     go1.26.6 고정.  안 건드린다
```

**SECURITY-10 이 이 줄로 준수다.** 잠금 파일이 커밋돼 있고 이 유닛이 그것을
안 움직인다.

### 2.2 새 저장소 0 — 이미 있는 디스크에 산다

```text
   자리     <Root>/progress/.  Root 는 cfg.Artifacts.Root 이고 record.Store 가
           이미 그 아래에 run-<id>/ 를 쓴다 (record.go:32)
   DB      스키마 diff 0.  진행 파일은 파일이지 행이 아니다
           (constraints.md 6절 「DB 스키마 변경 · 진행 로그는 디스크의 파일이다」)
```

**`execution-plan.md` 가 Infrastructure Design 을 SKIP 한 근거가 이것이다.**

### 2.3 새 실행파일 · 새 포트 · 새 전송 0

이 유닛은 네트워크를 **안 지난다.** HTTP 와 라우트는 U4 의 것이고 업로더는
U7 이다. `internal/record` 는 `net/http` 를 임포트하지 않고 이 유닛이 그것을
안 바꾼다.

### 2.4 새 설정 키 0 — V5 의 6시간을 상수로 둔다

```text
   근거 하나   decisions.md 2절의 상한 행이 「새 상한을 안 만든다」다
   근거 둘     이 값은 운영자가 조절할 값이 아니라 **보안 태세의 값**이다.
              인스턴스마다 다르면 requirements.md 5.4 의 잔여 ① 이 인스턴스마다
              다른 뜻이 된다
   근거 셋     선례가 전부 상수다 — 링 512 KiB (transcript.go:28) ·
              청크 2초 · 64 KiB (decisions.md 2절)
   되돌리는 값  실측이 값을 바꾸라고 하면 **상수 한 줄과 nfr-requirements.md 3.4** 를
              함께 고친다.  설정으로 빼는 것은 그때의 결정이다
```

### 2.5 새 타이머 0 — `RunReaper` 에 얹는다

```text
   쓰는 것    이미 도는 RunReaper.  주기는 cfg.Lease.RenewSeconds (기본 60초)
   실측       cmd/mediator/main.go:89 이 그 값으로 띄운다
             reap.go:158 의 time.NewTimer(0) 이 **기동에 한 번 먼저 돌게** 한다
   안 하는 것  새 고루틴.  새 주기 설정.  **주기를 두 벌로 두지 않는다**
```

**R24 가 요구한 「기동과 주기마다」를 기존 코드가 글자 그대로 만족한다.**
값을 고르는 대신 **있는 것을 찾았다.**

### 2.6 새 빌드 태그 0 — **들일 뻔했던 하나**

물음 4 가 이 자리였고 답이 **A (안 만든다)** 다.

```text
   B 였으면 들였을 것   디스크 여유를 재는 헬퍼.  오늘 그 코드가 실재한다 —
                     internal/enode/disk_unix.go 와 disk_windows.go 의
                     freeBytes 가 빌드 태그 쌍으로 갈려 있다

   왜 그대로 못 쓰나   **freeBytes 가 비공개다.**  그리고 record 가 enode 를
                     임포트하는 것은 방향이 거꾸로다 — enode 는 노드 에이전트이고
                     record 는 Mediator 의 저장 계층이다

   그래서 대가가       **같은 함수가 두 벌이 된다.**  CONVENTIONS.md 1.4 가
                     「도구는 두 벌로 두지 않는다 — 두 벌이 되면 규칙이 갈린다」로
                     막는 자리다.  피하려면 헬퍼를 공용 패키지로 옮기는
                     **별도의 일**이 생기고 그것은 이 유닛의 파일 행렬 밖이다

   의존은 문제가 아니었다   golang.org/x/sys 는 **이미 직접 의존**이다 (go.mod 실측).
                        B 를 골랐어도 go.mod 는 안 움직였다
```

**A 를 고른 대가는 `nfr-requirements.md` 5절의 R40 · R41 이다** — Mediator 가
자기 디스크를 안 보고, 그 결과 진행 파일이 봉인을 굶길 수 있다. **없앤 것이
아니라 받아들인 것이고 그 경로를 이름으로 적어 두었다.**

---

## 3. 이미 정해져 있어 고를 자리가 아니었던 것

**옮겨 적기를 안 한다** (`execution-plan.md` 의 「유닛마다 · 최소」). 가리키기만
한다.

```text
   상한 MaxBlobBytes 10 MiB     decisions.md 2절 · config.go:100
   청크 주기 2초 · 64 KiB       decisions.md 2절.  U7 의 것
   폴링 1초 · 2초 · 5초         requirements.md 5.7.  U5 · U8 의 것
   권한 0600 · 0700            requirements.md 5.4 · FD 의 R4
   파일 이름과 자리             D1 · FD 의 R1 · R2
   표시 줄의 모양               Q10 = B · FD 의 R12.  elidedMark 를 따른다
   언어와 툴체인                Go 1.26 · go1.26.6.  저장소 전체의 값이다
```

---

## 4. 저장 형식을 다시 고르지 않았다 — 근거

**「파일이 맞나」를 물을 수 있는 자리였고 묻지 않았다.** 코드가 답을 준다.

| 대안 | 왜 안 고르나 |
|---|---|
| DB 행 | `constraints.md` 6절이 **DB 스키마 변경을 명시로 뺐다.** 그리고 진행 파일은 순차 append 에 순차 read 라 관계형이 주는 것이 0 이다 |
| 링 파일 (노드의 `Ring` 처럼) | 링은 **고정 크기에 덮어쓴다.** 진행 파일은 `from` 오프셋으로 이어 읽어야 하고 (FR-5 · FR-6) 덮이면 그 오프셋이 뜻을 잃는다. 링이 푼 문제(윈도우의 파일 잠금)도 여기 없다 — Mediator 는 리눅스 하나다 |
| 메모리 | Mediator 가 재시작하면 사라진다. 노드가 총 길이로 오프셋을 맞추는 규약(FR-5)이 그 순간 깨진다 |
| 기존 `logs/` 에 그대로 | **질문 1 = B 가 이미 무른 자리다.** 파일을 갈라야 봉인 묶음에 원문이 안 들어간다 |

**append-only 평문 파일이 오늘 `logs/` 가 쓰는 것과 같은 형식이고**, 같은 것을
두 벌로 짓지 않는 것이 이 선택의 값이다.

---

## 5. 이 문서가 안 하는 것

```text
   N1 함대 규모      U4 의 것이다 (unit-of-work.md 9절)
   패턴 다섯        NFR Design 의 것이다
   시험 목록         Code Generation 의 것이다
   배포 · 운영       constraints.md 가 뺐다.  디스크 암호화도 거기다
                    (nfr-requirements.md 6.2 의 SECURITY-01 조건부)
```
