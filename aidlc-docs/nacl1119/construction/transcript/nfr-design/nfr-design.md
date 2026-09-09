# transcript — NFR Design

NFR Requirements 를 배선으로 확정한다. 정본 — 같은 폴더 nfr-requirements ·
`decisions.md` §6.3.

---

## 1. 단일 코드경로 (배선)

```text
   internal/enode/transcript.go   빌드 태그 없음.  os.File.WriteAt/ReadAt · encoding/binary 만.
                                  OS 분기 0 — Truncate·rename·삭제·잠금을 안 쓴다
   권한        OpenRing 이 0600 으로 만든다(유닉스).  윈도우는 모드 비트를 안 건다
              (policy.go·status.go 의 checkPerms 규약과 같은 태도)
```

---

## 2. 원자성 · 찢긴 읽기 (배선)

```text
   쓰기 순서    몸통을 WriteAt 한 뒤 머리의 total 을 WriteAt.  읽는 쪽이 total 을 커서로 신뢰
   읽기 재확인   ReadRing: 머리1 -> 몸통 -> 머리2.  total2-total1 이 capacity 이상이면
               몸통을 한 번 더 읽어 최근본으로 맞춘다(찢긴 읽기 완화)
   커서 밖 무시   total<=capacity 면 몸통[:total]만.  넘으면 start=total%capacity 부터 감아 읽는다
   세대         Reset 이 generation+1.  카드가 generation 변화로 화면을 비운다
```

머리를 마지막에 갱신하므로, 읽는 쪽이 몸통의 반쯤 쓰인 꼬리를 봐도 total 은 그
꼬리를 아직 안 가리켜 화면에 안 나온다.

---

## 3. tee 격리 (배선)

```text
   runner.go   cmd.Stdout = io.MultiWriter(&stdout, j.Transcript) — j.Transcript!=nil 일 때만.
              Decode·로그 반환은 그대로 &stdout.  링 쓰기 오류는 MultiWriter 가 반환하지만
              cmd.Run 은 이미 stdout 을 buffer 에도 쓰므로 실행은 계속 — 링은 보조다
   claim.go    w := io.MultiWriter(&buf, ring); cmd.Stdout,cmd.Stderr=w,w
   Worker      노드마다 OpenRing 한 번.  CLAIMED 시작 때 Reset.  못 열면(설정 없음/권한)
              nil 로 두고 진행 — 능력 저하일 뿐 실행을 막지 않는다
```

MultiWriter 는 한 writer 가 실패하면 멈추므로, ring.Write 는 실패해도 짧게
성공을 돌려주고 오류를 안 낸다(내부에서 삼키고 다음 쓰기를 이어간다) — tee 가
하네스 실행을 못 멈추게. 이 「삼킴」을 transcript.go 에 규칙으로 적는다.

---

## 4. 시험성 (배선)

```text
   internal/enode   링 파일: TempDir 에 OpenRing -> Write 로 감김·상한 초과·Reset·순서를 단언.
                    tee: 작은 Ring 에 stdout 을 흘려 ReadRing 이 그 바이트를 순서대로 내는지
   internal/panel   /api/transcript: TempDir 링을 만들고 GET 이 {generation,total,data} 를 내는지.
                    /api/runs·/api/record: httptest 가짜 Mediator + archive/tar 로 만든 tar 픽스처
```

---

## 5. 시각 디자인 (배선)

```text
   트랜스크립트 카드   design/enode-ux.pen S3(카드가 산다)·S4(자리만)를 Code Generation 에서
                     pencil MCP 로 읽어 지금 page.go 의 다크 테마·토큰에 맞춘다.
                     데몬 로그 카드 옆에 나란히 — 다른 물건임이 보이게 (CP6)
```

---

## 6. 안 넓히는 것

```text
   새 의존 0 · Mediator 변경 0 · 빌드 태그 쌍 0 · 하네스 계약 무변경 · 심볼 상한 무영향
```
