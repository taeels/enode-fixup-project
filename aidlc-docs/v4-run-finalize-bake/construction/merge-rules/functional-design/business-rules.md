# `merge-rules` — 규칙

대기 upper 를 lower 에 합칠 때 항목마다 무엇을 어떤 차례로 하나, 무엇을 보고 시작을 거절하나, 끊기면 어떻게 잇나의
규칙이다. 오류 문구는 영어다 (`CONVENTIONS.md` 2.1 — 밖으로 나간다). 타입과 이름은 `domain-entities.md` 에 있다.

**원칙 셋**

- **upper 가 곧 남은 일이다.** 표시(whiteout · opaque)는 그 효과를 lower 에 적용한 뒤에만 치운다 (결정 3-5 · ADR-077 §4).
  Apply 는 메모리에 아무것도 들고 가지 않는다 — 다시 부르면 upper 와 lower 와 5절의 표지만 보고 다시 판정한다
- **byte 를 옮기지 않는다.** 모든 옮기기는 같은 filesystem 안의 rename 이다. 비용은 upper 에서 바뀐 항목 수를 따르고,
  새 디렉터리는 하위 트리가 커도 rename 한 번이다 (`requirements.md` 5.4)
- **lower 를 바꾸는 호출은 lower 뿌리에서 디렉터리 fd 로 내려간 자리에서만 한다** (답 7 · 팩 보안 표의 합치기 줄). 예외는
  Discard 의 경로 하나다 (7절)

---

## 1. 표 — upper 항목마다 (ADR-077 §4 · 종류가 바뀐 항목 · 답 7)

차례는 왼쪽에서 오른쪽이다. 각 호출이 성공한 뒤 OnOp 가 불린다.

| upper 의 항목 | 같은 경로의 lower | 호출 | Result 칸 |
|---|---|---|---|
| 파일 · symlink · 그 밖 | 없다 | rename | Added |
| 파일 · symlink · 그 밖 | 파일 · symlink · 그 밖 | rename (대체) | Replaced |
| 파일 · symlink · 그 밖 | 디렉터리 | discard, rename | TypeChanged |
| 디렉터리 (opaque 아님) | 없다 | rename (덮어쓰지 않기) | NewDirs |
| 디렉터리 (opaque 아님) | 디렉터리 | stash-mtime (없을 때만), 안으로, chown, chmod, chtimes, rmdir | MergedDirs |
| 디렉터리 (opaque 아님) | 파일 · symlink · 그 밖 | discard, rename (덮어쓰지 않기) | TypeChanged |
| opaque 디렉터리 | 없다 | clear-opaque, rename (덮어쓰지 않기) | OpaqueDirs |
| opaque 디렉터리 | 있다 (무엇이든) | discard, clear-opaque, rename (덮어쓰지 않기) | OpaqueDirs |
| whiteout | 없다 | unlink | Whiteouts |
| whiteout | 있다 (무엇이든) | discard, unlink | Whiteouts |

- **lower 쪽은 lstat 으로 본다.** symlink 는 디렉터리를 가리켜도 「파일 · symlink · 그 밖」이다 — 그 안으로 들어가지 않고
  종류가 바뀐 항목으로 친다 (답 7 · 시제품의 `isdir and not islink` 와 같다)
- **종류가 바뀐 두 줄은 시제품이 한 그대로다** (계획 1.1 · FR-7). lower 쪽을 버린 뒤 옮긴다. 버리지 않고 옮기면 파일 위의
  디렉터리는 `ENOTDIR`, 디렉터리 위의 파일은 `EISDIR` 로 실패한다
- **파일 · symlink 를 옮기는 rename 은 대체가 뜻이다** — 보통 `renameat`. **디렉터리를 옮기는 rename 은 덮어쓰지 않기**
  (`renameat2` 의 `RENAME_NOREPLACE`)다. 보통 rename 은 lower 에 빈 디렉터리가 있으면 조용히 대체한다. 판정은 「없다」였는데
  실제로 무엇이 있으면 판정과 실제가 어긋난 것이고, 그때는 오류로 멈춘다
- **opaque 디렉터리는 lower 쪽 종류와 무관하게 한 줄이다.** lower 가 파일이어도 디렉터리여도 버린다
- **upper 뿌리는 표의 「양쪽 디렉터리」로 걷되, 속성을 맞추지 않고 지우지도 않는다** (계획 1.5 — 뿌리는 helper 가 0700 으로
  만든 것이다). stash-mtime 도 하지 않는다

---

## 2. 표시를 읽는 법 (답 8 · 계획 1.7)

```text
   whiteout      문자 장치 0/0 하나.  크기 0 파일에 붙은 user.overlay.whiteout 은 whiteout 으로 읽지 않는다 — 거절한다
   opaque        디렉터리의 user.overlay.opaque 값이 정확히 "y"
   이름          정확히 맞춘다.  앞부분 user.overlay. 로 보지 않는다
```

**있으면 시작하지 않는 것 (Preflight) · 걷다 만나면 그 항목 전에 멈추는 것 (Apply).** upper 뿌리를 포함한 모든 항목에서 본다.

```text
   user.overlay.metacopy           어느 항목에든               upper 파일이 내용 전체를 갖지 않는다 (ADR-077 §4)
   user.overlay.redirect           어느 항목에든               디렉터리 이름 바꿈을 해석하지 않는다 (ADR-077 §4)
   user.overlay.whiteout           어느 항목에든               overlayfs 는 만들지 않고, upper 층의 것은 커널이 반쪽만 읽는다
   user.overlay.opaque             값이 y 가 아니다 (x 포함)     x 는 opaque 가 아니다.  그 밖의 값은 뜻이 없다
                                   디렉터리가 아닌 항목에 있다
                                   upper 뿌리에 있다            lower 전체를 가리는 합치기는 없다
```

- 제품 마운트는 이것들을 만들지 않고, 세션 안의 프로세스도 못 만든다 — 세션이 붙인 `user.overlay.*` 는 upper 에
  `user.overlay.overlay.*` 로 적힌다 (계획 1.2 · 1.7). 있으면 그 upper 는 제품 마운트가 만든 것이 아니다
- **그대로 옮기는 것.** `user.overlay.origin` · `user.overlay.impure` 는 판정에 쓰지 않고 항목과 함께 lower 로 간다 (답 5 · 계획
  1.8 의 측정). escape 한 이름(`user.overlay.overlay.*`)도 보통 xattr 로 옮긴다 — 다음 overlay 가 한 겹 벗겨 세션에
  보여 준다. 그 밖의 `user.overlay.*` 이름도 판정에 쓰지 않고 옮긴다. 제품 마운트(index 꺼짐 · metacopy 꺼짐)에서
  origin · impure 말고 다른 이름이 항목에 붙는 것은 측정하지 못했다
- upper 뿌리의 xattr(예: `uuid=on` 이 남기는 것)은 lower 로 가지 않는다 — 뿌리는 옮기지 않는다

---

## 3. 시작 전 확인 — `Preflight` (FR-7 · 답 1 · 8)

싼 것부터 본다. 첫 어긋남에서 `*PreflightError` 를 돌려준다. 아무것도 바꾸지 않는다.

```text
   1  path         세 경로가 절대 경로다.  symlink 를 푼다 (filepath.EvalSymlinks) — 셋은 노드 설정이 준 경로다
   2  directory    푼 셋이 진짜 디렉터리다 (lstat).  trash 가 없어도 여기서 거절한다 — 부르는 쪽이 먼저 만든다
   3  overlap      푼 셋 중 어느 것도 다른 것과 같거나 그 안에 있지 않다 (경로 조각 단위로 앞부분을 비교한다)
   4  filesystem   셋의 st_dev 가 같다
   5  mark         upper 를 뿌리부터 디렉터리 fd 로 걸으며 항목마다 2절의 금지 표시를 찾는다
```

- **보지 않는 것 — 이 lower 의 overlay 마운트.** 부르는 쪽이 쥔 lower 배타 잠금이 「마운트 0」의 증거다 (답 1 · Application
  Design). 형제 helper 의 마운트는 호스트 mountinfo 에 안 보인다(계획 1.3). 그 측정과 옛 노드의 틈은 lower-state 유닛에
  넘긴다 (`business-logic-model.md` 7절)
- **같은 filesystem 이 아니면 시작하지 않는다.** 도중에 rename 이 `EXDEV` 로 실패하는 일을 앞에서 막는다
- **비용은 upper 항목 수를 따른다** (ADR-077 §12 — 15만 항목에 1.00 초 · 68만 항목에 3.95 초)
- 걷기의 이름 읽기(readdir)는 upper 디렉터리의 atime 을 바꿀 수 있다. 그래서 atime 은 맞추지 않는다 (5절)

---

## 4. `Apply` 의 걷기 (답 2 · 7 · 9)

- Discard 가 비어 있으면 아무것도 바꾸지 않고 `merge: no discard function` 을 돌려준다
- 뿌리 둘(푼 경로)을 `O_NOFOLLOW | O_DIRECTORY` 로 연다
- 디렉터리 안의 이름을 **바이트 순으로 정렬해** 차례로 본다. 같은 트리면 호출 차례가 같다 — 재개 시험의 N 이 하나로 정해진다
- 항목마다
  - upper 쪽 — `fstatat(부모 fd, 이름, AT_SYMLINK_NOFOLLOW)` 와 표시 읽기로 `classify`
  - lower 쪽 — 같은 이름을 lower 부모 fd 에서 `fstatat(AT_SYMLINK_NOFOLLOW)` 로 `LowerKind`. 없으면(`ENOENT`) LowerAbsent
  - `decide` 가 준 차례대로 호출한다 (1절의 표)
- 표시 읽기 — 디렉터리는 연 fd 로, 디렉터리가 아닌 항목은 `/proc/self/fd/<부모 fd>/<이름>` 에 l 계열 xattr 호출로 읽는다.
  경로를 이어 붙이지 않고 symlink 를 따라가지 않는다
- 양쪽 디렉터리면 두 쪽을 `openat(O_NOFOLLOW | O_DIRECTORY)` 로 열고 그 안에 같은 규칙을 적용한다. 속성 맞추기와 rmdir 는
  그 하위가 다 끝난 뒤에만 한다
- **끝나면 upper 뿌리가 비어 있어야 한다.** 아니면 `merge: upper is not empty after the walk` 로 돌려준다. 뿌리 자신은 지우지
  않는다 — 부르는 쪽이 치운다

---

## 5. lower 에도 있는 디렉터리 — 속성 (답 4)

```text
   맞추는 것    소유 (uid · gid) · mode (07777 — setuid · setgid · sticky 포함) · mtime
   차례         chown -> chmod -> chtimes.  chown 이 권한 비트를 바꿀 수 있어 chmod 가 뒤다.  시각은 마지막이다
   값의 출처    소유와 mode 는 맞출 때 upper 디렉터리를 fstat 한 값 — 옮기는 동안 바뀌지 않는다
               mtime 은 들어가기 전의 값 — user.enode.merge-mtime 에 적어 둔 것
   맞추지 않는 것  atime (lower 디렉터리의 것을 그대로 둔다 · UTIME_OMIT) · 뿌리
```

- **mtime 을 적어 두는 까닭.** 안의 항목을 옮기면 upper 디렉터리의 mtime 이 바뀐다. 한 번에 끝낼 때는 들어가기 전에 읽은
  값을 쓰면 되지만, 재개 때는 그 값이 이미 없다. 그래서 안의 항목에 처음 손대기 전에 upper 디렉터리 자신의 xattr
  `user.enode.merge-mtime` 에 10진 나노초로 적는다 (stash-mtime). 이미 있으면 적지 않고 그 값을 쓴다. 그 디렉터리는 rmdir
  로 사라지므로 표지가 lower 로 가지 않는다
  - 시제품은 적어 두지 않았고, 그래서 `listing.py` 가 디렉터리 mtime 을 비교에서 뺐다 (계획 1.1). 이 유닛은 재개 뒤에도
    디렉터리 mtime 이 한 번에 끝낸 것과 같다 — 시험이 디렉터리 mtime 까지 비교한다
- **atime 은 맞추지 않는다.** 시작 전 확인과 걷기의 이름 읽기가 upper 디렉터리의 atime 을 바꾸므로 옮길 값이 세션이 본 값이
  아니다. 시제품은 atime 도 옮겼다. ADR-077 §4 의 「시각」을 mtime 으로 읽는다
- 옮긴 항목(파일 · symlink · 통째로 옮긴 디렉터리)은 자기 inode 를 그대로 가지고 가므로 속성을 따로 맞추지 않는다. 디렉터리를
  다른 부모로 옮겨도 그 mtime 은 그대로다 (이 기계 ext4 에서 측정)
- **권한 비트를 풀었다 되돌리지 않는다.** merge-helper 는 namespace 안의 root 로 돈다 (`--map-root-user --map-auto` — 노드
  uid 와 subordinate 범위가 모두 매핑된다). 그 root 는 매핑된 항목이면 권한 비트와 무관하게 옮기고 지운다 — SunnyVM 에서
  0555 디렉터리 안으로 · 0555 · 000 디렉터리를 trash 로 옮기는 것을 측정했다 (노드 uid 로는 거부). 권한 비트를 풀었다
  되돌리면 그 사이에 끊겼을 때 lower 에 풀린 권한이 남는다
- **소유를 못 맞추면 멈춘다** — 매핑 밖 uid 면 `chown` 이 실패한다 (9절)

---

## 6. 끊김과 재개 (결정 3-5 · 답 2 · 9)

- **어느 호출 뒤에서든 끊길 수 있다.** 프로세스가 죽거나, ctx 가 끝나거나, 호출이 실패하거나, OnOp 가 오류를 돌려주거나
- **같은 절차를 다시 부르면 잇는다.** 이미 옮긴 것은 upper 에 없어 할 일이 없고, 이미 버린 lower 쪽은 없어서 「없다」줄로
  간다. 호출마다 다시 부르면 무엇을 하나는 `business-logic-model.md` 3절의 표다
- **오류 경로에서 되돌리지 않는다.** defer 로 치우거나 되돌리는 일을 두지 않는다 — 오류로 멈춘 자리와 죽어서 멈춘 자리가
  같은 모양으로 남는다. 열어 둔 fd 를 닫는 것만 한다
- **Discard 는 lower 쪽 항목 하나를 한 번만 버린다.** 버린 뒤에는 lower 에 없으므로 재개가 다시 버리지 않는다

---

## 7. 경로 경계 (답 7 · 팩 보안 표의 합치기 줄)

- 세 뿌리 아래에서 symlink 를 따라가지 않는다 — `O_NOFOLLOW` · `AT_SYMLINK_NOFOLLOW` · l 계열 xattr
- 바꾸는 호출(rename · unlink · rmdir · xattr 쓰기와 지우기 · chown · chmod · 시각)의 대상은 모두 뿌리에서 fd 로 내려간
  디렉터리 바로 아래의 이름이다. 이름은 이름 읽기가 준 것이라 `/` 가 없고, `.` · `..` 는 건너뛴다
- **예외 하나 — Discard 의 경로.** 푼 lower 뿌리에 상대 경로를 이어 `scratch.Trash.Move` 에 넘긴다 (답 6). 그 위의 조각은
  걷기가 방금 진짜 디렉터리로 확인했고, 배타 잠금 아래라 사이에 바꿀 프로세스가 없다. `Trash.Move` 의 rename 은 마지막
  조각이 symlink 여도 링크 자체를 옮긴다
- 같은 filesystem 이 아니면 시작하지 않는다 (3절)

---

## 8. 결과 세기 (답 9)

- `Ops` 는 성공한 호출 수다. 실패한 호출은 세지 않는다
- 항목 칸(일곱 중 하나)은 그 항목의 마지막 호출(rename · unlink · rmdir)이 성공했을 때 는다. `Discarded` 는 discard 가
  성공할 때마다 는다
- **Result 는 그 호출이 한 일만 센다.** 재개로 다시 부르면 남은 일만 센다 — 끊기기 전의 몫과 더하는 것은 부르는 쪽이다

---

## 9. 오류와 멈춤 (답 9)

- **첫 오류에서 멈춘다.** 호출이 실패하면 `*OpError`(호출 이름과 상대 경로)로 감싸 돌려준다. Discard 의 오류도 같다
- **ctx 가 끝나면 다음 호출 전에 멈춘다** — 호출 하나가 끝난 뒤, OnOp 뒤에 본다. `ctx.Err()` 를 돌려준다
- **OnOp 의 오류는 감싸지 않고 그대로 돌려준다** — 시험이 자기 오류를 알아보게
- 돌려주는 Result 는 그때까지 센 것이다
- **남은 upper 가 곧 남은 일이다.** 멈춘 자리보다 위의 디렉터리는 속성도 맞추지 않고 지우지도 않은 채 남는다
- **같은 오류가 재개 때마다 되풀이될 수 있다** (예: 매핑 밖 uid 로 chown). 그러면 상태가 merging 에 머문다 — 그때 무엇을
  하나는 bake 유닛에 넘긴다

---

## 10. 오류 문구 (영어)

```text
   Preflight
     merge preflight: <root>: path is not absolute
     merge preflight: <root>: <원래 오류>                                    풀 수 없음 · lstat 실패
     merge preflight: <root>: not a directory
     merge preflight: <a> and <b> overlap
     merge preflight: upper, lower and trash must share one filesystem (upper dev <n>, lower dev <n>, trash dev <n>)
     merge preflight: upper entry <rel> carries user.overlay.metacopy
     merge preflight: upper entry <rel> carries user.overlay.redirect
     merge preflight: upper entry <rel> carries user.overlay.whiteout; only 0/0 character devices are read as whiteouts
     merge preflight: upper entry <rel> has user.overlay.opaque="<값>"; only "y" on a directory is read as opaque
     merge preflight: upper entry . carries user.overlay.opaque; the upper root cannot be opaque

   Apply
     merge: no discard function
     merge: <call> <rel>: <원래 오류>                                        예: merge: rename d1/a.txt: ...
     merge: upper entry <rel> carries ...                                  걷다 만난 금지 표시 — 위 Preflight 의 문구와 같은 뒷부분
     merge: upper is not empty after the walk
     merge is supported on linux only                                      linux 밖
```

`<rel>` 은 upper 뿌리 기준 상대 경로이고, 뿌리 자신은 `.` 이다. 상대 경로를 쓰는 까닭 — 절대 경로는 노드의 host 경로라
Run 의 로그로 나갈 때 길고 겹친다. 뿌리 경로는 부르는 쪽이 이미 안다.

---

## 11. 코드 경계 시험 (유닛 정의 5절)

`internal/panel/boundary_test.go` 의 표에 더한다.

```text
   금지 하나   cmd/mediator 는 internal/merge 를 못 가져다 쓴다
   봉인 하나   internal/merge 는 표준 라이브러리와 golang.org/x/sys 만 쓴다 (internal/scratch 도 안 된다 — 답 6)
```

---

## 12. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 합치기 줄은 3절(같은 filesystem)과 7절(경로가 lower 밖으로 안 나감)이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 표 시험과 가짜 트리 재개 시험 (저장소 관례) |
