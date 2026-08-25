package enode

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// R5②' — git 이 안 보는 산출물을 걷는다
//
// `git status` 는 .gitignore 를 지킨다. 그게 diff 를 만들 때는 값이지만
// (데워둔 빌드 캐시가 안 섞인다), 훅이 산출물을 검토시킬 때는 정확히 반대다 —
// 빌드 산출물이 전부 무시 목록에 있어서 훅이 zImage 도 .ko 도 못 본다.
//
//	git diff        추적 파일의 변경   → 사람이 리뷰할 것           (diff.go)
//	이 파일           무시되는 것 포함    → 훅이 검토시킬 것
//
// 왜 mtime 인가
//
//	✗ fanotify  FAN_MARK_FILESYSTEM 이 CAP_SYS_ADMIN 을 요구한다.
//	            개발자 컨테이너 안에서는 못 쓴다.
//	✗ inotify   디렉터리마다 watch 를 걸어야 한다. 커널 트리는 수천 개고
//	            빌드 중 새로 생기는 디렉터리를 따라가야 한다.
//	✗ syscall 가로채기  seccomp user-notify 로 되긴 하지만(실측) openat 이
//	            수백만 번이라 빌드가 못 돌 만큼 느려진다.
//	○ 기준 시각 + walk 한 번        특권 없음 · 어디서나 됨 · 비용이 유계다.

// changedName 은 변경 요약이 실리는 산출물 이름이다.
//
// diff 와 이름이 다른 이유 — 보는 것이 다르다.
//
//	workspace.diff      추적 파일의 내용 — 사람이 리뷰한다
//	workspace.changed   무시되는 것 포함 목록 — 훅이 검토시키고 사람이 확인한다
const changedName = "workspace.changed"

// Stamp 는 「이 시각 이후에 생긴 것」의 기준이다.
//
// 파일이 아니라 시각을 들고 다닌다 — 워크스페이스에 도장 파일을 남기면
// 그것 자체가 산출물로 걷히고 sanitize 대상이 된다.
type Stamp struct {
	At   time.Time
	Root string
}

// stampNow 는 지금을 기준으로 잡는다. sanitize 직후에 부른다 —
// 그래야 「이 단계가 만든 것」과 「원래 있던 것」이 갈린다.
//
// 1초를 뺀다. 파일시스템 mtime 해상도가 초 단위인 경우가 있어,
// 같은 초에 만들어진 파일을 놓치는 것보다 조금 더 걷는 편이 낫다.
func stampNow(root string) Stamp {
	return Stamp{At: time.Now().Add(-time.Second), Root: root}
}

// Changed 는 한 파일의 변경이다.
type Changed struct {
	Path string // 워크스페이스 기준 상대경로
	Size int64
}

// changedSince 는 기준 시각 이후에 생기거나 바뀐 파일을 모은다.
//
// .git/ 은 뺀다 — 인덱스·로그가 계속 바뀌어 목록을 덮는다. 그건 산출물이 아니다.
// 상한을 두는 이유는 커널 빌드가 수만 개를 만들기 때문이다. 넘으면
// 개수만 세고 목록은 자른다 — 잘랐다는 사실을 숨기지 않는다.
func changedSince(s Stamp, limit int) (found []Changed, total int, err error) {
	if s.Root == "" {
		return nil, 0, nil
	}
	// 상한을 정렬 앞에 걸면 안 된다 — 먼저 만난 것만 남아 vmlinux 가 빠진다.
	// 그래서 일단 모으고, 정렬한 뒤 자른다. 커널 빌드 10만 개라도 경로+크기라
	// 수 MB 이고 단계 끝에 한 번이라 유계다.
	const ceiling = 200000
	var all []Changed
	err = filepath.WalkDir(s.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 읽을 수 없는 곳은 건너뛴다. 여기서 단계를 죽이지 않는다.
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".repo":
				return filepath.SkipDir
			}
			return nil
		}
		fi, err := d.Info()
		if err != nil || !fi.ModTime().After(s.At) {
			return nil
		}
		total++
		if len(all) < ceiling {
			rel, _ := filepath.Rel(s.Root, p)
			all = append(all, Changed{Path: rel, Size: fi.Size()})
		}
		return nil
	})
	// 큰 것이 앞에 온다 — 최종 산출물(vmlinux · zImage · .ko)이 대개 크고,
	// 중간 부산물(.o · .cmd · .d)은 작다. 우리가 무엇이 산출물인지 고르지는
	// 않지만, 모델이 먼저 볼 것을 정해줄 수는 있다.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Size != all[j].Size {
			return all[i].Size > all[j].Size
		}
		return all[i].Path < all[j].Path
	})
	if len(all) > limit {
		all = all[:limit]
	}
	return all, total, err
}

// summarize 는 목록을 모델이 읽을 수 있는 크기로 줄인다.
//
// 커널 빌드는 수만 개를 만든다. 그대로 넣으면 컨텍스트가 터지고, 터지지 않더라도
// 모델이 못 읽는다. 확장자별로 세고, 큰 것 몇 개를 이름으로 보여준다.
//
// 개수가 적은 확장자가 보통 최종 산출물이다 (.ko 12개 vs .o 4231개).
// 그렇다고 우리가 골라주지 않는다 — 무엇이 산출물인지는 계약과 모델이 안다.
// 훅은 알리기만 한다 (판정은 success_when 이 한다 — ADR-004 · I3).
func summarize(found []Changed, total, topN int) string {
	if total == 0 {
		return ""
	}
	byExt := map[string]int{}
	for _, c := range found {
		e := strings.ToLower(filepath.Ext(c.Path))
		if e == "" {
			e = "(no extension)"
		}
		byExt[e]++
	}
	type kv struct {
		e string
		n int
	}
	var exts []kv
	for e, n := range byExt {
		exts = append(exts, kv{e, n})
	}
	sort.Slice(exts, func(i, j int) bool {
		if exts[i].n != exts[j].n {
			return exts[i].n > exts[j].n
		}
		return exts[i].e < exts[j].e
	})

	var b strings.Builder
	fmt.Fprintf(&b, "files created or modified by this step: %d", total)
	if len(found) < total {
		fmt.Fprintf(&b, " (showing %d)", len(found))
	}
	b.WriteString("\n  by extension: ")
	for i, e := range exts {
		if i >= 8 {
			fmt.Fprintf(&b, " and %d more", len(exts)-8)
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s %d", e.e, e.n)
	}
	b.WriteString("\n  largest first:\n")
	for i, c := range found {
		if i >= topN {
			break
		}
		fmt.Fprintf(&b, "    %-52s %s\n", c.Path, human(c.Size))
	}
	return b.String()
}

func human(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// writeStamp / readStamp — 훅은 별도 프로세스라 기준 시각을 넘겨받아야 한다.
//
// 워크스페이스 밖에 둔다 — 안에 두면 자기가 걷히고 sanitize 에 지워진다.
func writeStamp(path string, s Stamp) error {
	return os.WriteFile(path, []byte(s.At.Format(time.RFC3339Nano)), 0o600)
}

func readStamp(path, root string) (Stamp, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Stamp{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b)))
	if err != nil {
		return Stamp{}, err
	}
	return Stamp{At: t, Root: root}, nil
}

// CheckChanged 는 계약이 지목한 경로들이 이 단계 안에 바뀌었는지 본다 (ADR-037).
//
// 판정하지 않는다 — 바뀐 것만 돌려주고 대조는 Mediator 가 한다
// (ADR-005 조립자=평가자). 노드는 판정 조건을 모른다.
//
// 전체 목록을 걷지 않는다 — 커널 빌드는 수만 개를 만든다.
// 지목된 경로만 stat 하므로 비용이 계약이 적은 만큼이다.
func CheckChanged(stamp Stamp, want []string) []string {
	if stamp.Root == "" || len(want) == 0 {
		return nil
	}
	var got []string
	for _, rel := range want {
		fi, err := os.Stat(filepath.Join(stamp.Root, rel))
		if err != nil || fi.IsDir() {
			continue // 없거나 디렉터리면 「바뀌었다」가 아니다
		}
		// 기준 시각 이후에 쓰였는가 — changedSince 와 같은 판정이다.
		if !fi.ModTime().Before(stamp.At) {
			got = append(got, rel)
		}
	}
	return got
}
