package enode

import (
	"os"
	"testing"
)

// TestMachineAttributeSaysWhichMachine 은 같은 기계의 노드를 셀 수 있는지를 잰다.
//
// hostname 은 node_id 안에 해시로만 있어서 계약도 매처도 못 읽는다.
// 그것이 안 보이면 기계마다 굽는 창을 어긋나게 둘 수 없다 (ADR-070 §2.2 · §5.5).
func TestMachineAttributeSaysWhichMachine(t *testing.T) {
	host, err := os.Hostname()
	if err != nil || host == "" {
		t.Skip("this machine has no hostname to compare against")
	}
	attrs := cheapAttrs(Local{}, testLog())
	if attrs["machine"] != host {
		t.Errorf("machine is %q, want %q", attrs["machine"], host)
	}
}

// TestMachineIsNotACapability 는 machine 하나만으로 노드가 광고하지 않는지를 잰다.
//
// os · host_arch · ws 와 같은 자리다 — 어느 기계에나 있는 사실이지 능력이
// 아니다. 능력으로 세면 아무것도 할 줄 모르는 기계가 agent.reason 을
// 광고하고, 계약이 그것을 집어 실행 시점에 죽는다.
func TestMachineIsNotACapability(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // 툴체인도 하네스도 없는 기계

	l := Local{}
	log := testLog()
	if caps := capabilities(l, log, nil, cheapAttrs(l, log)); len(caps) != 0 {
		t.Errorf("a machine that can do nothing advertised %d capabilities: %v", len(caps), caps)
	}
}

// TestLabelsCannotOverrideMachine 은 사람이 적은 것이 탐지를 못 이기는지를 잰다.
func TestLabelsCannotOverrideMachine(t *testing.T) {
	host, err := os.Hostname()
	if err != nil || host == "" {
		t.Skip("this machine has no hostname to compare against")
	}
	l := Local{Labels: map[string]string{"machine": "someone-elses-box"}}
	log := testLog()
	caps := capabilities(l, log, map[string]string{"harness": "claude"}, cheapAttrs(l, log))
	if len(caps) == 0 {
		t.Fatal("a node with a harness must advertise")
	}
	if got := caps[0].Attrs["machine"]; got != host {
		t.Errorf("a label overrode detection: machine is %q, want %q", got, host)
	}
}
