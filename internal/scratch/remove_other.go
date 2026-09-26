//go:build !unix

package scratch

import "errors"

// errUnsupported 는 trash 삭제가 없는 OS 의 답이다 — runc-overlay 노드가 linux 에만 있다.
var errUnsupported = errors.New("trash removal is not supported on this platform")

// Measure 는 이 OS 에서 지원하지 않는다.
func Measure(string, string) (Size, error) { return Size{}, errUnsupported }

// Remove 는 이 OS 에서 지원하지 않는다.
func Remove(string, string) ([]string, error) { return nil, errUnsupported }
