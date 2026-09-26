//go:build !linux

package merge

import "context"

// Preflight 는 linux 밖에서 지원하지 않는다 — 합치기는 runc-overlay 노드(linux)만 한다.
func Preflight(Paths) error { return ErrUnsupported }

// Apply 는 linux 밖에서 지원하지 않는다.
func Apply(context.Context, Paths, Options) (Result, error) { return Result{}, ErrUnsupported }
