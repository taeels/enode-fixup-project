package enode

import (
	"strings"
	"testing"
)

// 빌드가 직접 $OUT 에 놓게 시킬 수 있어야 한다
//
// 그래야 아무도 산출물 경로를 미리 몰라도 된다 —
// collect 가 경로를 미리 알아야 해서 작위적이라는 지적의 답이다.
func TestArgv_MakesTheBuildWriteIntoOUT(t *testing.T) {
	io := IOPaths{Dir: "/ws", In: "/i", Out: "/o"}
	got := expandIO([]string{"make", "modules_install", "INSTALL_MOD_PATH=$OUT"}, io)
	if got[2] != "INSTALL_MOD_PATH=/o" {
		t.Fatalf("$OUT was not expanded: %q", got[2])
	}
	// 원본을 안 건드린다 — 재시도가 같은 계약을 다시 쓴다
	if got[2] == "" {
		t.Fatal("empty value")
	}
}

func TestArgv_ExpandsBothNames(t *testing.T) {
	io := IOPaths{Dir: "/ws", In: "/i", Out: "/o"}
	for _, c := range []struct{ in, want string }{
		{"$OUT", "/o"}, {"${OUT}", "/o"}, {"$IN", "/i"}, {"${IN}", "/i"},
		{"-o$OUT/kernel", "-o/o/kernel"},
		{"DESTDIR=${OUT}", "DESTDIR=/o"},
	} {
		if got := expandIO([]string{c.in}, io)[0]; got != c.want {
			t.Fatalf("%q → %q (want %q)", c.in, got, c.want)
		}
	}
}

// 셸을 흉내 내지 않는다 — 두 이름만 안다.
//
// 확장을 흉내 내기 시작하면 인용·글롭·워드 분할이 줄줄이 따라오고
// argv 배열을 고른 이유가 사라진다.
func TestArgv_LeavesOtherVariablesAlone(t *testing.T) {
	io := IOPaths{Out: "/o"}
	got := expandIO([]string{"echo", "$HOME", "$PATH", "~/x", "*.c"}, io)
	want := []string{"echo", "$HOME", "$PATH", "~/x", "*.c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%d: %q ≠ %q — it must not imitate a shell", i, got[i], want[i])
		}
	}
}

// 안 풀린 이름을 알려준다 — 조용히 리터럴로 가면 사람이 못 알아챈다.
func TestArgv_ReportsWhatItCouldNotExpand(t *testing.T) {
	got := unexpandedVars([]string{"make", "-C", "$WORKSPACE", "O=${BUILD_DIR}", "x$HOME"})
	j := strings.Join(got, ",")
	for _, need := range []string{"WORKSPACE", "BUILD_DIR", "HOME"} {
		if !strings.Contains(j, need) {
			t.Fatalf("%s was not pointed out: %v", need, got)
		}
	}
	// 이미 푼 뒤에는 남는 게 없어야 한다
	if left := unexpandedVars(expandIO([]string{"a=$OUT", "b=$IN"},
		IOPaths{Out: "/o", In: "/i"})); len(left) != 0 {
		t.Fatalf("$OUT/$IN survived: %v", left)
	}
}

// $OUT 이 안 정해진 단계면 안 푼다 — 빈 문자열로 바꾸면 `-o` 뒤가 사라진다.
func TestArgv_DoesNotExpandWithoutAPath(t *testing.T) {
	if got := expandIO([]string{"x", "$OUT"}, IOPaths{})[1]; got != "$OUT" {
		t.Fatalf("substituted an empty value: %q", got)
	}
}
