# panel — NFR Requirements

이 유닛의 비기능 요구다. 대부분 팩이 값으로 못 박았고(심볼 상한 · 커버리지 ·
바인딩/토큰 · 크로스빌드), 이 문서는 그 요구와 **수용 기준**을 모은다. 값을
새로 정하는 자리는 하나뿐이다(폴링 간격 — §2, 팩 권장을 그대로 딛는다).

정본 — `enode-features.md` 3.1.2 비기능 · `constraints.md`(구조 불변식 · 심볼
상한 · 커버리지) · `decisions.md` §2·§3 · `.github/workflows/ci.yml` 차단 스텝 ·
`requirements/scene-gates.md` CP4.

---

## 1. 성능 · 바이너리 (차단 게이트)

```text
   요구                                          수용 기준
   ───────────────────────────────────────────   ────────────────────────────────────────
   enodectl.exe 심볼 상한                          GOOS=windows 빌드에서 net/http T 심볼 <= 50 ·
   (ci.yml:479~489 · continue-on-error 없음)        crypto/tls T 심볼 <= 10.  지금 6 · 1.
                                                  serve 의 exec 위임과 proc 의 net/http 무의존으로 지킨다.
                                                  유닛 종료 시 재측정해 상한 안임을 적는다

   제어판 응답성                                   단일 사용자 · 로컬 루프백이라 처리량 목표가 없다.
                                                  화면은 값 대신 「마지막 갱신 시각」을 보인다
                                                  (팩 권장 · design/README §5)
```

**심볼 상한이 이 유닛의 유일한 성능성 차단 게이트다.** 앞 회차에 `enodectl.exe`
가 `net/http`·`crypto/tls` 를 통째로 링크해 백신이 지운 사건이 근거다
(`scripts/avprobe/README.md`). NFR Design 이 그 회피(exec 위임 · proc 분리)를
설계로 확정한다.

---

## 2. 폴링 · 갱신 (팩 권장을 딛는다)

```text
   현재 작업 조회   Mediator 폴링.  팩 권장 5초 (decisions §2 「폴링 간격」)
   탐지 능력       상태 파일 읽기.  데몬이 At 바뀔 때만 쓰므로 제어판은 화면 열림 동안 읽는다
   화면 표기       채워지는 진행 막대가 아니라 「마지막 갱신 시각」 (scene-gates §2.1 · design §5)
```

값을 새로 안 만든다 — 팩이 5초로 권장했고 그대로 딛는다. transcript 유닛의
링 파일 1초 폴링은 이 유닛 밖이다.

---

## 3. 보안 (security-baseline · 차단)

```text
   요구                                          수용 기준
   ───────────────────────────────────────────   ────────────────────────────────────────
   바인딩 기본 127.0.0.1:8081                      기본 바인딩으로 띄우면 다른 기계에서 접속 안 됨
   LAN 노출은 --listen 명시로만                     --listen 이 127.0.0.1 밖인데 panel_token 이 비면
   (SECURITY-07 · 08)                              New(cfg) 가 error · serve 가 안 뜬다 (거부 경로)
   Mediator 토큰 재사용 금지 (SECURITY-12)          panel_token 은 정책 파일에만.  브라우저·화면에 실 토큰 없음
   파일 권한                                       정책·상태 파일 유닉스 0600 · 윈도우 상속 ACL.
                                                  남이 쓸 수 있으면 경고 (policy.go checkPerms 규약)
   기본 무인증의 근거                               그 기계에 접속한 것이 소유의 증거 (ADR-063 §3).
                                                  읽기 전용 카드는 편집 자리를 안 만든다 (ADR-017 결정 3)
```

이 회차엔 LAN 을 안 켠다 — **거부 경로만 선다**(decisions 「LAN 노출의 토큰」).
켜는 것은 한 줄이고, 뒤 회차의 몫이다.

---

## 4. 신뢰성 · 저하 (graceful degradation)

제어판은 무상태다. 의존이 죽어도 나머지는 산다 — business-rules §2 의 분기가
이 요구를 화면으로 편다.

```text
   데몬이 죽음        프로세스 status=stopped · start(S5) 가 보인다.  로컬 파일은 그대로 답한다
   Mediator 불통      로컬 묶음(신원·탐지 능력·프로세스·drain)은 그대로.
                     「Mediator 마지막 응답 N초 전」과 drain 통보 지연 문구를 보인다
   상태 파일 없음     탐지 능력 「아직 모름」.  데몬이 못 써도 능력은 광고가 진다 — 막지 않는다
```

---

## 5. 시험성 · 커버리지 (차단)

```text
   요구                          수용 기준
   ───────────────────────────   ─────────────────────────────────────────────
   패키지 커버리지 하한 80%        internal/panel · internal/proc 둘 다.  ci.yml:269 의 awk floor=80
   DB 없이 도는 테스트            제어판·proc 는 Postgres 를 안 탄다 — 파일과 fake HTTP 로 채운다
   스킵 0                        .ci-allowed-skips 허용목록에 새 스킵을 안 넣는다
   임포트 경계 검사 테스트         panel -> store/api 금지 · enode -> panel 금지를 테스트로 잡는다
```

`internal/proc` 는 플랫폼 짝이라 유닉스·윈도우 각 빌드 태그의 테스트가 자기
쪽만 돈다 — 커버리지는 각 OS 빌드에서 잰다.

---

## 6. 이식성 · 빌드 (차단)

```text
   크로스 빌드      linux · darwin · windows 셋 다 go build 통과 (ci.yml cross 잡)
   빌드 태그 짝      proc_unix.go · proc_windows.go 를 internal/proc 로 옮겨도 짝이 유지된다
   OS별 권한        상태·정책 파일 권한이 유닉스(0600)와 윈도우(상속 ACL)로 갈린다 — 코드가 가른다
```

---

## 7. 기술 스택 · 의존 (무변경)

```text
   HTTP 서버        표준 라이브러리 net/http 만.  새 웹 프레임워크 0
   YAML            상태·정책 파일은 gopkg.in/yaml.v3 — 이미 있는 의존이다 (policy.go)
   새 Go 의존       0.  go.mod 무변경 · packaging/ 무변경 (새 실행파일 없음 · decisions §1)
```

---

## 8. 표기 · 언어 (차단)

```text
   glyphscan       비테스트 Go 파일 문자열 리터럴에 장식 문자 0 (ci.yml · CONVENTIONS §1)
   언어            에러·로그·CLI·테스트 이름/픽스처는 영어.  주석·문서는 한국어 (CONVENTIONS §2)
```

---

## 9. 이 단계에서 안 정하는 것

```text
   위 회피를 어떻게 배선하나       NFR Design 몫 (exec 위임 · proc 분리 · 상태 파일 쓰기 자리)
   화면 레이아웃                 시안 design/*.pen · 진행자
   LAN 토큰을 켜는 것            뒤 회차.  이 회차는 거부 경로만
```
