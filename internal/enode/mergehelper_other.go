//go:build !linux

package enode

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/taeels/enode/internal/merge"
)

// errMergeHelperLinux 는 merge-helper 가 없는 OS 의 답이다 — 굽기는 runc-overlay 노드 (linux) 만 한다.
var errMergeHelperLinux = errors.New("merge-helper is supported on linux only")

// RunMergeHelper 는 이 OS 에서 할 일이 없다.
func RunMergeHelper(_ io.Reader, _, errOut io.Writer) int {
	fmt.Fprintln(errOut, errMergeHelperLinux)
	return 1
}

// callMergeHelper 는 helper 를 못 띄운다는 답만 한다. Baker 는 linux 에서만 만들어진다.
func callMergeHelper(mergeHelperRequest) (merge.Result, error) {
	return merge.Result{}, errMergeHelperLinux
}

// fileOwner 는 linux 밖에서 모른다 — 초안 읽기가 거절한다.
func fileOwner(os.FileInfo) (int, bool) { return 0, false }
