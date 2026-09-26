# `merge-rules` — Code Generation 요약

**유닛** `merge-rules` (합치기 규칙) · **브랜치** `unit/merge-rules` · **기준** `f155faf` (Functional Design 커밋) ·
**계획** `construction/plans/merge-rules-code-generation-plan.md` (열두 단계) · **맡는 조각** 7 의 기계 부분

새 패키지 `internal/merge` 가 대기 upper 를 lower 에 합친다 — `Preflight` 와 `Apply` 두 함수다. 부르는 쪽(merge 단계 ·
merge-helper · 시작 때 재개)은 bake 유닛이 짓는다. 지금은 어느 바이너리도 이 패키지를 링크하지 않는다.

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/merge/merge.go` | 새 | 패키지 문서 · `Paths` · `Options` · `Call` 아홉과 `String` · `Op` · `Result` · `tally` · `Check` 다섯 · 오류 타입 셋 · `ErrUnsupported` · `stashXattr` |
| `internal/merge/decide.go` | 새 | 판정 (모든 플랫폼) — `Kind` · `LowerKind` · `Marks` · `splitNames` · `marksFrom` · `classify` · `lowerKindOf` · `step` · `decide` · `checkOverlap` · `checkDevices` · `joinRel` |
| `internal/merge/merge_linux.go` | 새 | `Preflight` · `Apply` — 디렉터리 fd 로 걷는 호출 · 표시 읽기 · 재개 표지 |
| `internal/merge/merge_other.go` | 새 | linux 밖 — `ErrUnsupported` |
| `internal/merge/decide_test.go` | 새 | 판정 표 시험 — 모든 플랫폼 |
| `internal/merge/tree_linux_test.go` | 새 | 가짜 트리 · 목록 · 모형 `view` · 시험의 Discard · `verify` |
| `internal/merge/merge_linux_test.go` | 새 | 한 번에 · 재개 41 자리 · ctx · Discard 오류 · Discard 없음 · 금지 표시에서 멈춤 · 덮어쓰지 않기 · 새 디렉터리 한 번 · 나쁜 표지 · symlink 뿌리 · Preflight 표 · 벤치마크 둘 |
| `internal/merge/merge_integration_test.go` | 새 | `integration` 태그 · linux · CI 밖 — 진짜 overlay 로 |
| `internal/panel/boundary_test.go` | 고침 | 금지 `{"cmd/mediator", "internal/merge"}` · 봉인 `{"internal/merge", golang.org/x/sys/}` |

계획 2절 밖의 파일은 없다. 시험이 바꾼 `cmd/enodectl/probe.lock` 은 되돌렸다.

---

## 2. 규칙의 자리

```text
   FD 규칙 1절 (표 열 줄)                decide · walker.item
   FD 규칙 2절 (표시를 읽는 법 · 금지)       marksFrom · classify · marksOfDir · marksAt
   FD 규칙 3절 (시작 전 확인 다섯)          Preflight · resolve (1 ~ 4) · scan · scanDir (5)
   FD 규칙 4절 (걷기)                     Apply · walker.walk · readNames (바이트 순)
   FD 규칙 5절 (양쪽 디렉터리의 속성)        walker.mergeDir · readStash
   FD 규칙 6절 (끊김과 재개)               walker.call — 되돌리는 defer 가 없다
   FD 규칙 7절 (경로 경계)                 openDirFlags (O_NOFOLLOW) · Fstatat(AT_SYMLINK_NOFOLLOW) · *at 호출 · Discard 의 경로
   FD 규칙 8 · 9절 (결과 · 오류와 멈춤)     walker.call · Result.count · OpError
   FD 규칙 10절 (문구)                    PreflightError · MarkError · OpError 의 Error · errNoDiscard · errNotEmpty
   FD 규칙 11절 (경계 시험)                internal/panel/boundary_test.go
```

---

## 3. 계획 4절의 결정이 어떻게 들어갔나

- **① 디렉터리가 아닌 항목의 표시** — `marksAt` 가 `/proc/self/fd/<부모 fd>/<이름>` 에 `Llistxattr` · `Lgetxattr` 를 부른다. symlink · 장치
  파일은 빈 목록이 돌아온다
- **② 호출 자리의 함수 칸** — 두지 않았다. 가짜 트리만으로 89.1% 다 (5절)
- **③ 시험의 Discard** — `trashRec.discard` 가 시험 trash 로 옮긴다 (이름은 순번). `internal/scratch` 를 쓰지 않는다
- **④ 재개의 끊기** — OnOp 가 k 번째에 `errStop` 을 돌려주고, Apply 가 그것을 그대로 돌려준다(`errors.Is`). k 는 1 .. 41 (가짜 트리의 `Ops`)
- **⑤ 목록** — `entry` 는 경로 · 종류 · 권한 07777 · uid · gid · 크기 · mtime (파일과 디렉터리) · symlink 과녁 · inode (디렉터리가 아닌 항목)
- **⑥ integration** — 자기 바이너리를 `unshare --user --map-root-user --map-auto --mount` 뒤에서 다시 실행한다. 아이가 권한 000 · 0555 를
  남긴 폴더를 `t.TempDir` 이 지울 수 있게 풀고 끝낸다
- **⑦ 벤치마크의 트리** — `BenchmarkApply` 는 반복마다 트리를 짓고 그 시간을 뺀다
- **⑧ 문구** — FD 규칙 10절 그대로다. `decide_test.go` 의 `TestErrorMessages` 가 문장마다 본다

---

## 4. 계획 · FD 와 다른 자리

- **`classify` 의 인자.** FD 엔티티는 `classify(mode, rdev, marks, root bool)` 이었다. 코드는 `classify(rel, mode, rdev, marks)` 다 —
  `*MarkError` 에 경로를 담아야 하고, 뿌리는 `rel == "."` 로 안다
- **읽기 오류의 문구.** FD 규칙 10절은 바꾸는 호출의 오류(`merge: <call> <rel>: ...`)만 적었다. 읽다 실패한 것은 호출이 아니므로
  `merge: read <rel>: ...` · `merge: read lower <rel>: ...` · `merge: open upper|lower: ...` 로 낸다. Preflight 에서 읽다 실패한 것은
  `merge preflight: read <rel>: ...` 이고 **`*PreflightError` 가 아니다** — 확인이 어긋난 것이 아니라 확인을 못 한 것이다 (8절 bake)
- **재개 표지가 읽히지 않으면 멈춘다** — `merge: read <rel>: bad user.enode.merge-mtime "<값>"`. FD 에 없던 갈래다
- **integration 시험의 `mv`.** 세션 안의 `os.Rename` 은 lower 디렉터리에 `EXDEV` 를 돌려준다. 시험은 `mv` 가 그 뒤에 하는 일 —
  지우고 새로 만들기 — 을 한다. upper 에 남는 표시는 같다 (whiteout 하나와 새 디렉터리)
- **계획에 없던 확인 — 변이 넷.** 코드를 일부러 틀리게 고쳐 시험이 빨개지는지 보았다. 모두 잡혔고 되돌렸다
  - 재개 표지를 무시한다 → 재개 시험이 여러 자리에서 실패 (디렉터리 mtime)
  - opaque 를 lower 를 버리기 **전에** 지운다 → 한 번에 시험은 통과하고 **재개 시험의 16번째 자리 하나만** 실패. 호출마다 끊는
    시험(답 2)이 있어야 잡히는 틀림이다
  - lower 쪽을 `stat` 으로 본다 (symlink 를 따라간다) → 한 번에 시험 실패
  - `internal/merge` 가 `internal/scratch` 를 임포트한다 → 경계 시험 실패

---

## 5. 코드 검사 (CI 와 같은 명령)

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go vet -tags integration ./internal/merge/` · `go build ./...` | exit 0 |
| U+2605 · `go run ./scripts/glyphscan.go` | 0 · 148 파일에 장식 문자 없음 |
| `go test ./... -count=1` (`-coverpkg=./...` · `-json` · 시험 DB · 가짜 claude 스텁) | 통과 2,178 · 실패 0 · 스킵 0 |
| 패키지마다 커버리지 (CI 의 awk 그대로) | 전부 80% 이상 · 전체 86.4%. 새 `internal/merge` 89.1% (303/340) · `internal/panel` 89.1% 그대로 |
| 크로스 빌드 셋 | windows/amd64 · linux/arm (GOARM=7) · darwin/arm64 exit 0. linux/mips 에서도 vet 이 통과한다 (`Rdev` 가 uint32 인 자리) |
| `golangci-lint run ./...` (v2.13.2) | 38 건 — Step 1 과 같다 |

`internal/merge` 의 함수 — 판정 쪽(`decide.go` · `merge.go`)은 100%. `merge_linux.go` 는 `Apply` 78.4% · `xattrValue` 73.3% ·
`xattrNames` 76.9% · 나머지 80 ~ 93.5%. 남은 것은 열기 · 읽기 실패와 `ERANGE` 다시 묻기 같은 오류 갈래다.

**조각 0** — build · vet · test 초록. **조각 7 의 기계 부분** — `TestApplyResume` 이 기본 `go test` 에서 41 자리 모두 초록.

---

## 6. 측정 — NFR 을 건너뛰어 계획 3절이 받은 것

**보안.** 가짜 트리의 lower `escape` 는 lower 밖 디렉터리를 가리키는 symlink 이고 그 위에 upper 디렉터리가 온다. 합친 뒤 밖 디렉터리가
그대로이고(`verify` 가 매번 본다 · 재개 41 자리 포함) lower 의 `escape` 는 진짜 디렉터리다. 시작 전 확인의 겹침 · 같은 filesystem 은
표 시험이 본다.

**성능 — byte 와 무관.** 한 번에 시험이 새 파일 · 대체한 파일 · 하드 링크의 inode 가 upper 의 것인지 본다 (복사하지 않았다).
파일 1,000 개의 새 디렉터리는 `Ops` 1 이다 (`TestApplyNewDirOnce`).

**성능 — 하루치 규모.** 벤치마크 둘, 3회 평균.

```text
                    항목                              이 기계 (N100 4코어 · VM)   SunnyVM (8코어 · ext4)   ADR-077 §12 (시제품 · SunnyVM)
   Preflight        upper 150,100 항목                  0.63 초                   0.23 초                1.00 초 (147,893 파일)
   Apply            58,423 항목 · 호출 72,485            3.80 초                   1.11 초                1.49 초 (연산 59,030)
```

- SunnyVM 에서 둘 다 시제품보다 빠르다. 같은 기계 · 같은 규모라 §12 와 댈 수 있다
- 이 기계가 느린 것은 입출력이다. 프로파일(트리 짓기 포함)에서 CPU 표본은 벽시계의 46% 이고, 그 대부분이 시스템 호출(rename · 만들기)이다.
  `Llistxattr` 는 약 4% 다 — `/proc/self/fd` 경로로 읽는 비용은 작다
- 호출 수 72,485 가 §12 의 59,030 보다 큰 것은 단위가 달라서다 (FD 엔티티 2절 — 양쪽 디렉터리 하나가 다섯 번)
- 값은 트리를 막 지은 뒤라 page cache 가 따뜻한 상태의 것이다. §12 와 같은 조건이다

---

## 7. integration 시험 — SunnyVM (커널 7.0) · 2026-09-26 초록

에이전트가 SunnyVM 의 ext4 에 둔 버려도 되는 폴더(`TMPDIR=~/merge-it-tmp`)에서 돌렸고, 끝나고 바이너리 · 로그 · 폴더를 지웠다.

```text
   세 목록         커널 merged view = 모형 view = 합친 lower     같다
   Result         Ops 44 · Replaced 2 · Added 3 · NewDirs 2 · OpaqueDirs 3 · Whiteouts 3 · TypeChanged 1 · MergedDirs 4 · Discarded 7
   권한           lower ro · rodeep · rodeep/sub 이 0555 그대로.  권한 000 디렉터리의 whiteout 도 합쳐졌다
   다음 overlay    합친 lower 위의 새 overlay 가 lower 와 같은 목록을 보이고, 덧붙이기(다시 복사해 올리기)가 된다
   재개           k = 1 .. 44 모두 같다
```

- 커널은 lower symlink(`linkdir`)를 지우고 같은 이름으로 만든 디렉터리를 **opaque 로** 만들었다. 가짜 트리의 같은 자리는 opaque 가
  아닌 디렉터리(종류가 바뀐 항목)다 — 두 줄을 모두 시험한 셈이다
- 이 결과로 사람 조각(조각 7 전체 · bake 유닛)을 대신하지 않는다

---

## 8. 다른 유닛에 넘기는 것 (FD 흐름 7절 그대로 · 이 단계가 더한 것 하나)

```text
   lower-state   「마운트 0 의 증거」 — 계획 1.3 의 측정 · 옛 노드의 틈

   bake          merge-helper 안에서 Preflight 뒤 Apply — 처음과 재개 모두 같은 두 줄
                 Options.Discard 를 scratch.Trash.Move 로.  Preflight 전에 trash 를 만든다
                 merge-helper 는 namespace 안의 root 로 돈다.  권한을 내려놓지 않는다
                 PreflightError.Check 마다 상태를 어디로 돌리나
                 (더함) Preflight 가 *PreflightError 가 아닌 오류를 돌려주면 확인을 못 한 것이다 — 읽기 실패.
                   어긋난 것과 달리 다시 해 볼 일이다
                 같은 오류가 재개 때마다 되풀이될 때 · 빈 upper 뿌리를 치우기 · Result 를 더하기 (칸이 바뀔 수 있다)
                 한 합치기의 trash 항목을 한 디렉터리에 모을지 (하루치 526 번 · helper 한 번에 약 5 ms)
```

---

## 9. 정본에 되돌려 올리는 것

FD 흐름 8절 그대로다 (진행자가 `enode-design` 에 올린다). 이 단계가 더하는 사실 하나 — ADR-077 §12 옆에 제품 코드의 값을 적을 수 있다:
SunnyVM 에서 같은 규모의 Preflight 0.23 초 · Apply 1.11 초 (호출 72,485).
