//go:build !linux

package enode

import (
	"context"
	"log/slog"
)

// overlayFP 는 리눅스 밖에서는 아무것도 광고하지 않는다.
//
// overlayfs 는 리눅스의 것이다. 맥과 윈도우 노드는 추론과 빌드를 하지
// 오버레이 워크스페이스를 나눠 쓰지 않는다 — 속성이 빠지는 것이 곧
// "못 한다" 이므로 (ADR-012) 그 노드는 오버레이를 요구하는 계약의 후보에서
// 그냥 빠진다.
type overlayFP struct{}

func (overlayFP) Kind() string { return "overlay" }

func (overlayFP) Probe(_ context.Context, _ Local, _ *slog.Logger) (map[string]string, error) {
	return nil, nil
}
