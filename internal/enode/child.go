package enode

import (
	"os/exec"
	"time"
)

// waitDelay 는 취소된 뒤 파이프가 닫히기를 기다리는 최대 시간이다.
//
// 자식을 죽여도 손자가 남으면 Wait 가 안 돌아온다 — 부모의 stdout 파이프를
// 손자가 상속했으면 그 쓰기 끝이 안 닫히고, 우리는 Stdout 을 버퍼로 받으므로
// Run 이 복사 고루틴을 기다린다. 그 사이 부르는 쪽(광고 루프)은 여전히 멈춰 있다.
//
// 실측 (2026-09-05) — `#!/bin/sh` 에 `sleep 60` 하나만 담은 가짜 하네스로
// Detect 를 부르고 200ms 뒤 컨텍스트를 취소했더니, 취소가 CommandContext 까지
// 닿는데도 Detect 가 10초를 넘겨 안 돌아왔다. ctx 를 넘기는 것만으로는
// 부족하고 이 값이 있어야 실제로 손을 뗀다.
//
// 1초인 이유 — 정상 종료 중인 자식의 마지막 출력을 자르지 않을 만큼 길고,
// 광고 주기(기본 60초)에 비하면 무시할 만큼 짧다.
const waitDelay = time.Second

// child 는 우리가 띄우는 모든 외부 프로그램이 지나는 한 곳이다.
//
// 이 패키지가 외부 프로그램을 띄우는 자리는 여덟 곳이고 앞으로 더 는다.
// 자리마다 손으로 설정을 붙이면 언젠가 하나를 빠뜨리는데, 빠뜨린 자리는
// 윈도우에서만, 혹은 자식이 걸렸을 때만 티가 나므로 평소에는 안 보인다.
// 그래서 한 곳으로 모으고, child_test.go 가 감싸지 않은 자리를 센다.
//
// 두 가지를 건다.
//
//	콘솔을 안 물려준다     왜 필요한지는 console_windows.go 의 childAttr 이 적는다
//	자식보다 오래 안 산다   왜 필요한지는 위의 waitDelay 가 적는다
func child(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = childAttr()
	cmd.WaitDelay = waitDelay
	return cmd
}
