# `merge-rules` — 흐름

타입은 `domain-entities.md`, 규칙과 문구는 `business-rules.md` 에 있다. 여기는 그 규칙이 어느 차례로 도나, 끊긴 자리마다
다시 부르면 무엇을 하나, 조각 7 의 기계 부분을 어떻게 확인하나, 그리고 다른 유닛에 무엇을 넘기나다.

---

## 1. 부르는 쪽과의 약속 (bake 유닛의 merge 단계 · 시작 때 재개)

이 유닛은 아래 흐름에서 `Preflight` 와 `Apply` 두 줄만 만든다. 나머지는 bake · lower-state 유닛의 몫이다.

```text
   부르는 쪽 (노드 · bake 유닛)                              internal/merge
   ----------------------------------------------------      ----------------------------------------------
   lower 배타 잠금을 쥔다 (lower-state)
     -> 「이 lower 의 overlay 마운트 0」의 증거 (답 1)
   상태를 merging 으로
   <scratch>/trash 를 만든다 (0700)
   merge-helper 를 연다 — unshare --user --map-root-user
     --map-auto (runtime · trash-helper 와 같은 매핑)
     Preflight(paths)                                  ->   확인 다섯.  아무것도 안 바꾼다
       어긋나면 *PreflightError                           <-
     Apply(ctx, paths, Options{Discard: trash.Move})   ->   걷기 (2절)
       Result, err                                     <-
   err 가 없으면  빈 upper 뿌리를 치운다 · metadata (FR-9) · committed
   err 가 있으면  merging 에 둔다.  다음 시작 때 같은 둘을 다시 부른다
```

- **처음과 재개가 같은 두 줄이다.** 재개는 끊긴 자리를 몰라도 된다 — 남은 upper 가 그 자리다
- **Preflight 와 Apply 는 helper 안에서 돈다.** upper 에 노드 uid 가 못 걷는 항목(subordinate uid 소유 · 권한 000)이 생길 수
  있고 (`components.md` 6절), 권한 비트가 없는 디렉터리를 옮기는 일은 namespace 안의 root 만 한다 (`business-rules.md` 5절)

---

## 2. `Apply` 의 걷기

```text
   Apply(ctx, p, opt)
     opt.Discard 가 없으면 -> 오류 (아무것도 안 바꿈)
     upperRoot, lowerRoot 를 연다 (푼 경로 · O_NOFOLLOW | O_DIRECTORY)
     walk(upperRoot, lowerRoot, ".", root = true)
     upperRoot 가 비어 있나 -> 아니면 오류
     return result

   walk(u, l, rel, root)
     names = u 의 이름들, 바이트 순
     for name in names
       uk = classify(fstatat(u, name), marks(u, name), false)        금지 표시면 멈춤
       lk = lowerKind(fstatat(l, name))
       s  = decide(uk, lk)
       if s 가 「안으로」
         mergeDir(u, l, name, rel/name)
         continue
       if s.discard      -> call(Discard(lowerRoot/rel/name))
       if s.clear        -> call(fremovexattr(u/name, user.overlay.opaque))
       if s.last == Rename -> call(renameat 또는 renameat2 NOREPLACE (u, name, l, name))
       if s.last == Unlink -> call(unlinkat(u, name))
       tally(s)

   mergeDir(u, l, name, rel)
     ud = openat(u, name) · ld = openat(l, name)
     mtime = getxattr(ud, user.enode.merge-mtime)
     없으면 -> mtime = fstat(ud).mtime · call(fsetxattr(ud, user.enode.merge-mtime, mtime))
     walk(ud, ld, rel, false)
     st = fstat(ud)
     call(fchown(ld, st.uid, st.gid))
     call(fchmod(ld, st.mode & 07777))
     call(utimensat(l, name, atime = UTIME_OMIT, mtime))
     call(unlinkat(u, name, AT_REMOVEDIR))
     tally(MergedDirs)

   call(x)
     x 를 한다.  실패 -> *OpError 로 멈춤
     result.Ops++ · OnOp(op) 가 오류면 그대로 멈춤 · ctx 가 끝났으면 ctx.Err() 로 멈춤
```

- **뿌리는 `walk` 만 한다.** `mergeDir` 을 지나지 않으므로 stash-mtime · 속성 맞추기 · rmdir 이 없다 (계획 1.5)
- **뿌리의 표시도 본다.** 뿌리에 opaque 가 있으면 멈춘다 — Preflight 가 먼저 거절한다 (`business-rules.md` 2절)
- 열어 둔 fd 는 깊이마다 둘이다. yocto 트리의 깊이(수십)에서 fd 수는 문제가 되지 않는다

---

## 3. 끊긴 자리마다 다시 부르면 (결정 3-5 · 답 2)

「끊긴 자리」는 그 호출이 성공한 뒤다. 다시 부른 Apply 는 그 항목을 처음부터 다시 판정한다.

| 줄 | 끊긴 자리 | 다시 부르면 보이는 것 | 다시 부른 Apply 가 하는 일 | 결과 |
|---|---|---|---|---|
| 파일이 lower 디렉터리 위 | discard 뒤 | upper 파일 · lower 없음 | rename | 같다 (Added 로 센다) |
| 디렉터리가 lower 파일 위 | discard 뒤 | upper 디렉터리 · lower 없음 | rename (덮어쓰지 않기) | 같다 (NewDirs 로 센다) |
| opaque 가 lower 위 | discard 뒤 | upper opaque · lower 없음 | clear-opaque, rename | 같다 |
| opaque | clear-opaque 뒤 | upper 보통 디렉터리 · lower 없음 | rename (새 디렉터리 줄) | 같다 (NewDirs 로 센다) |
| whiteout 이 lower 위 | discard 뒤 | upper whiteout · lower 없음 | unlink | 같다 |
| 양쪽 디렉터리 | stash-mtime 뒤 | 표지 있음 | 표지의 mtime 을 쓴다 · 안으로 | 같다 |
| 양쪽 디렉터리 | 안의 어느 호출 뒤 | 안에 남은 upper | 표지의 mtime · 남은 것만 | 같다 |
| 양쪽 디렉터리 | chown · chmod · chtimes 뒤 | 빈 upper 디렉터리 · 표지 있음 | 속성을 다시 맞춘다 (같은 값) · rmdir | 같다 |
| 어느 줄이든 | 마지막 호출(rename · unlink · rmdir) 뒤 | upper 에 그 항목이 없다 | 할 일이 없다 | 같다 |

- **결과는 lower 와 trash 가 같다는 뜻이다.** 끊지 않은 한 번과 비교해 lower 의 목록(디렉터리 mtime 포함)이 같고, 버린 lower
  쪽 항목의 모임이 같고, upper 뿌리가 비어 있다
- **Result 칸은 다를 수 있다.** 끊긴 뒤의 한 번은 남은 일만 세고, 위 표처럼 칸이 바뀌기도 한다 (종류가 바뀐 항목이 재개 때
  Added 로 세어진다). 합친 수를 보고할 때 이것을 안다 (7절 bake)

---

## 4. 조각 7 의 기계 부분 — 시험의 모양 (답 2 · 3)

### 4.1 기계 (기본 `go test` · CI 에서 돈다)

**가짜 트리.** 특권 없이 만든다 — whiteout 은 `mknod c 0 0`, opaque 는 `setxattr user.overlay.opaque y` (계획 1.4). 모든 디렉터리는
쓰기 가능이다 (namespace 가 없다). 표의 줄마다 하나 이상을 담는다.

```text
   lower                               upper                                          표의 줄
   a.txt                               a.txt (내용 다름)                               파일 대체
   (없음)                              new.txt                                        새 파일
   link -> a.txt                       link -> new.txt                                symlink 대체
   d1/ b.txt c.txt sub/ s.txt          d1/ (mode 700 · mtime 다름) c.txt whiteout ·     양쪽 디렉터리 · 안의 whiteout ·
                                         sub/ (mtime 다름) t.txt                        두 겹 속성 맞추기
   d2/ x/ y.txt                        d2/ opaque · z.txt                             opaque 가 lower 디렉터리 위
   d3/ deep/ q.txt                     d3 whiteout · d3moved/ deep/ q.txt              디렉터리 whiteout · 새 디렉터리
   f_to_dir (파일)                      f_to_dir/ opaque · g.txt                        opaque 가 lower 파일 위
   f_to_plain (파일)                    f_to_plain/ (opaque 아님) h.txt                 종류가 바뀐 항목 — 디렉터리가 파일 위
   d_to_file/ inner                    d_to_file (파일)                                종류가 바뀐 항목 — 파일이 디렉터리 위
   linkdir -> d1                       linkdir/ (opaque 아님) m.txt                    lower symlink 는 디렉터리가 아니다 (답 7)
   (없음)                              brandnew/ sub/ k.txt                           새 디렉터리 하위 트리째
   (없음)                              wh_absent whiteout                             lower 없는 whiteout
   (없음)                              opq_absent/ opaque · o.txt                      lower 없는 opaque
   keep/ untouched.txt                 (없음)                                          손대지 않는 lower
   (없음)                              hl1 · hl2 (한 inode)                            하드 링크 둘 — 둘 다 같은 inode 로
   (없음)                              esc.txt · user.overlay.overlay.opaque = y       escape 한 이름은 옮긴다 (2절)
   a.txt 의 형제에                      origin.txt · user.overlay.origin                origin 은 그대로 옮긴다 (답 5)
```

**시험 여섯.**

```text
   classify · decide      표 시험 — 종류 넷 x lower 셋 · 금지 표시 다섯 (metacopy · redirect · xattr whiteout · opaque x ·
                          디렉터리가 아닌 항목의 opaque) · 뿌리의 opaque · escape 한 이름은 표시가 아니다
   한 번에                Apply 한 번 -> lower 목록 = 모형 view(합치기 전 lower, upper) · upper 뿌리가 비었다 ·
                          버린 lower 쪽 항목의 inode 모임이 모형의 「가려진 lower」와 같다 · 새 파일의 inode 가 upper 의 것이다
                          (복사하지 않았다) · Result 칸 · Ops = N
   재개 (N 번 모두)        k = 1 .. N 마다 트리를 새로 짓고 OnOp 가 k 번째에 오류 -> 그 오류가 그대로 온다 -> 다시 Apply ->
                          「한 번에」와 lower 목록 (디렉터리 mtime 포함) · 버린 모임 · 빈 upper 가 같다
   ctx                    k 번째 OnOp 에서 cancel -> ctx.Err() -> 다시 Apply -> 같다
   오류                   Discard 가 처음 한 번 실패 -> *OpError (discard · 경로) -> 다시 Apply -> 같다
   Preflight              표 — 상대 경로 · 없는 trash · 파일인 lower · 겹침 셋 (같음 · 안 · 밖) · 금지 표시 다섯 · 뿌리 opaque ·
                          통과하는 트리 (origin · escape 한 이름 · 0/0 whiteout 만 있음).  st_dev 판정은 세 값을 받는
                          순수 함수로 표 시험 (다른 filesystem 을 시험 안에서 만들 수 없는 CI 가 있다)
```

- **N 은 가짜 트리의 「한 번에」가 센 Ops 다.** 팩의 「1 ~ 30」을 넘어 마지막 호출까지 모두 끊는다 (답 2)
- **가짜 표시를 못 만들면 시험이 실패한다** — 건너뛰지 않는다 (유닛 정의 5절). CI 의 `ubuntu-latest` 에서 되는지는 이 유닛의
  첫 CI 가 확인한다 (계획 1.4). 못 만들면 그 시험의 자리를 다시 정한다 (`requirements.md` 5.6)
- **모형은 Apply 와 다른 길로 짠다** (`domain-entities.md` 6절). 모형을 커널에 대 보는 것은 4.2 다

### 4.2 사람이 SunnyVM 에서 도는 시험 (`integration` 태그 · CI 밖)

시험 바이너리를 `unshare --user --map-root-user --map-auto` 뒤에서 다시 실행한다 (helper 와 같은 매핑). 버려도 되는 자리에서만
돈다 — `/srv/yocto` 는 건드리지 않는다 (`requirements.md` 5.5).

```text
   진짜 표시     lower 를 짓고 진짜 overlay 세션 안에서 toy-test.sh 의 동작을 한다 — 덧붙이기 · 새 파일 · rm · chmod ·
                rm -rf 뒤 mkdir (opaque) · mv (EXDEV 복사) · 파일 자리에 디렉터리 · 디렉터리 자리에 파일 · ln -sfn ·
                읽기 전용(0555) 디렉터리 안에 새 파일 · 000 디렉터리를 rm -rf
   세 목록       커널 merged view (세션 안에서 읽은 목록 · inode 뺌) = 모형 view = 합친 lower  (답 3)
   재개          같은 트리로 k = 1 .. N 끊고 다시 -> 같다
   다음 overlay  합친 lower 위에 새 overlay 를 올려 목록이 lower 와 같다 · 덧붙이기가 된다 (계획 1.8 을 시험으로)
   권한          0555 · 000 디렉터리가 권한 비트를 안 바꾼 채 합쳐진다 (business-rules.md 5절)
```

기본 `go test` 가 못 보는 것이 셋이다 — 커널이 만든 진짜 표시, namespace 안의 권한, 읽기 전용 디렉터리. 이 시험의 결과로
사람 조각(조각 7 전체 · bake 유닛)을 대신하지 않는다.

---

## 5. 순수 함수와 커버리지

새 패키지는 80% 이상이어야 한다 (`unit-of-work-file-matrix.md` 4절). 측정 플랫폼은 linux/amd64 다 (`.coverage-contract.yml`).

```text
   모든 플랫폼   classify · decide · Call.String · 오류 타입 셋의 Error · Unwrap         표 시험
   linux        Preflight · Apply 의 걷기와 호출                                      가짜 트리 시험 (4.1) — 특권 없이 돈다
   linux 밖      ErrUnsupported 두 줄                                                  linux/amd64 프로파일에 안 나온다
```

`_linux.go` 의 오류 갈래 중 가짜 트리로 못 밟는 것(예: `openat` 실패)은 남는다. 80% 를 넘지 못하면 Code Generation 이 호출
자리를 함수 값으로 떼어 실패를 끼운다.

---

## 6. 파일 행렬

행렬(`unit-of-work-file-matrix.md`)이 이 유닛에 준 자리는 `internal/merge/` (새) 와 `internal/panel/boundary_test.go` 다.
그 밖의 파일을 만지지 않는다. `boundary_test.go` 는 trash 유닛도 만진 파일이라 main 을 받은 뒤 줄을 더한다.

---

## 7. 다른 유닛에 넘기는 것

```text
   lower-state   「마운트 0 의 증거」를 닫을 때 받을 것 둘 (답 1)
                   계획 1.3 의 측정 — 형제 helper 의 overlay 마운트는 호스트 mountinfo 에 안 보이고 /proc/<pid>/mountinfo 에서 보인다
                   옛 노드의 틈 — lower-state 전의 노드는 잠금을 안 쥔다 (SunnyVM 의 yocto 노드가 2026-09-23 빌드로 떠 있다)

   bake          merge-helper 안에서 Preflight 뒤 Apply — 처음과 재개 모두 같은 두 줄 (1절)
                 Options.Discard 를 scratch.Trash.Move 로 채운다 (trash 유닛이 넘긴 약속 그대로 · 답 6).  Preflight 전에 trash 를 만든다
                 merge-helper 는 namespace 안의 root 로 돈다 — 권한 비트를 안 바꾸고 0555 · 000 을 옮기는 데 기댄다.
                   권한을 내려놓지 않는다
                 PreflightError.Check 마다 상태를 어디로 돌리나 (bake FD 의 물음 「시작 전 확인이 어긋났을 때」)
                 같은 오류가 재개 때마다 되풀이될 때 — 상태가 merging 에 머문다 (business-rules.md 9절)
                 Apply 가 끝나면 빈 upper 뿌리를 치운다
                 Result 를 로그와 metadata 에 — 재개면 끊기기 전과 뒤의 Result 를 더한다.  칸이 바뀔 수 있다 (3절)
                 한 합치기의 trash 항목을 한 디렉터리에 모을지 — 삭제자가 항목마다 helper 를 연다 (하루치 526 번 · 약 5 ms 씩)

   이 유닛의 NFR  보안 — business-rules.md 3절 · 7절이 닫은 것을 확인한다
                 성능 — 하루치 규모의 upper 에서 Preflight 와 Apply 의 시간 (ADR-077 §12 의 1.00 초 · 1.49 초와 대조).
                 Ops 는 호출마다라 §12 의 연산 수보다 크다 (domain-entities.md 2절)
```

---

## 8. 정본과 회차 문서에 되돌려 올리는 것

**정본** — 진행자가 `enode-design` 에 올린다.

```text
   ADR-077 §4    표에 종류가 바뀐 항목 두 줄 (lower 쪽을 버린 뒤 옮긴다 · lower 의 symlink 는 디렉터리가 아니다)
                 「커널이 xattr 형식을 쓰는 경우도 whiteout 으로 읽는다」를 고친다 — overlayfs 는 xattr whiteout 을 만들지 않고,
                   upper 층의 것은 커널이 반쪽만 읽는다 (목록에 있고 열면 없다).  합치기는 거절한다.  opaque 는 값 y 만이다
                 lower 에도 있는 디렉터리의 「시각」은 mtime 이다.  재개 때를 위해 들어가기 전에 적어 둔다.  atime 은 안 맞춘다
                 origin · impure 는 lower 로 따라가도 다음 overlay 가 정상이다 (측정)
                 디렉터리를 옮기는 rename 은 덮어쓰지 않기다
                 「합치는 동안 이 lower 의 overlay 마운트 0」의 증거는 부르는 쪽의 배타 잠금이다
   ADR-077 §12   시제품의 「연산」은 항목마다 셌다 (trash 옮기기만 따로).  제품의 Ops 는 호출마다다
```

**회차 문서** — 이 단계의 커밋이 함께 고친다 (답 8 이 「유닛 정의 5절을 고친다」를 골랐다).

```text
   unit-of-work.md 5절        「whiteout 은 문자 장치 0/0 과 xattr 형식」 -> 문자 장치 0/0.  xattr 형식은 거절
   components.md 2.2          「whiteout 읽기 — 문자 장치 0/0 과 xattr 형식」 -> 같게
   component-methods.md 2절   Classify 의 주석 -> 같게.  겉면이 바뀐 것은 domain-entities.md 1절이 적는다
```

---

## 9. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 합치기 줄은 `business-rules.md` 3절 · 7절이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 표 시험과 가짜 트리 재개 시험 |

다음 단계는 유닛 정의대로면 이 유닛의 NFR Requirements (최소 — 보안 · 성능)다.
