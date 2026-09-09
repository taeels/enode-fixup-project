package enode

import (
	"path/filepath"
	"strings"
	"testing"
)

func openRing(t *testing.T, capacity int) (*Ring, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ws-a.transcript")
	r, err := OpenRing(p, capacity)
	if err != nil {
		t.Fatalf("OpenRing: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, p
}

func TestTranscriptPath(t *testing.T) {
	if got := TranscriptPath("/etc/enode/ws-a.yaml"); got != "/etc/enode/ws-a.transcript" {
		t.Fatalf("TranscriptPath = %q", got)
	}
}

func TestRingWriteRead(t *testing.T) {
	r, p := openRing(t, 64)
	if _, err := r.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}
	snap, err := ReadRing(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(snap.Data) != "hello world" {
		t.Fatalf("data = %q", snap.Data)
	}
	if snap.Total != 11 {
		t.Fatalf("total = %d", snap.Total)
	}
}

func TestRingWrapsAtCapacity(t *testing.T) {
	r, p := openRing(t, 16)
	// 20 bytes into a 16-byte body: oldest 4 are overwritten, last 16 remain in order.
	if _, err := r.Write([]byte("0123456789abcdefghij")); err != nil {
		t.Fatal(err)
	}
	snap, _ := ReadRing(p)
	if string(snap.Data) != "456789abcdefghij" {
		t.Fatalf("wrapped data = %q (want 456789abcdefghij)", snap.Data)
	}
	if snap.Total != 20 {
		t.Fatalf("total = %d", snap.Total)
	}
}

func TestRingWrapsAcrossBoundary(t *testing.T) {
	r, p := openRing(t, 10)
	r.Write([]byte("aaaaaaa")) // 7 bytes, cursor at 7
	r.Write([]byte("bbbbbb"))  // 6 bytes: 3 fill 7..9, 3 wrap to 0..2
	snap, _ := ReadRing(p)
	// total=13 > cap=10, start=13%10=3, body = [b b b a a a a b b b], read from 3: aaaabbbbbb... let us just assert last-10 order
	if len(snap.Data) != 10 {
		t.Fatalf("len = %d", len(snap.Data))
	}
	if string(snap.Data) != "aaaabbbbbb" {
		t.Fatalf("boundary data = %q (want aaaabbbbbb)", snap.Data)
	}
}

func TestRingResetClears(t *testing.T) {
	r, p := openRing(t, 64)
	r.Write([]byte("old step output"))
	if err := r.Reset(); err != nil {
		t.Fatal(err)
	}
	snap, _ := ReadRing(p)
	if len(snap.Data) != 0 {
		t.Fatalf("after reset data = %q (want empty)", snap.Data)
	}
	gen := snap.Generation
	r.Write([]byte("new step"))
	snap2, _ := ReadRing(p)
	if string(snap2.Data) != "new step" {
		t.Fatalf("after reset+write = %q", snap2.Data)
	}
	if snap2.Generation != gen {
		t.Fatalf("generation should not change on write: %d vs %d", snap2.Generation, gen)
	}
	if gen == 0 {
		t.Fatal("generation should have bumped on reset")
	}
}

func TestRingReopenKeepsState(t *testing.T) {
	r, p := openRing(t, 64)
	r.Write([]byte("persisted"))
	_ = r.Close()
	r2, err := OpenRing(p, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close() //nolint:errcheck
	snap, _ := ReadRing(p)
	if string(snap.Data) != "persisted" {
		t.Fatalf("reopened data = %q", snap.Data)
	}
	// a further write continues from the persisted cursor
	r2.Write([]byte("!"))
	snap2, _ := ReadRing(p)
	if string(snap2.Data) != "persisted!" {
		t.Fatalf("appended data = %q", snap2.Data)
	}
}

func TestRingWriteIsBestEffort(t *testing.T) {
	// Write must never return an error, so io.MultiWriter never aborts cmd.Run.
	r, _ := openRing(t, 8)
	n, err := r.Write([]byte(strings.Repeat("x", 100)))
	if err != nil || n != 100 {
		t.Fatalf("Write = %d, %v (want 100, nil)", n, err)
	}
}
