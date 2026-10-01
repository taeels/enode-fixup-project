package scratch

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func i64(v int64) *int64 { return &v }

func at(t time.Time) *time.Time { return &t }

// 규칙 2절의 ④ — 여유 하한을 먼저 보고, 몫은 (보존 + 여유) 의 비율이다. 여유를 못 쟀으면 거르지 않는다.
func TestDecideAdmission_Order(t *testing.T) {
	p := Policy{CapacityPercent: 20, MinFree: 10 * gib}
	cases := []struct {
		name   string
		a      Admission
		ok     bool
		reason string
		detail string
	}{
		{"unknown free space", Admission{Kept: 100 * gib}, true, "", ""},
		{"room and nothing kept", Admission{Free: 100 * gib, FreeKnown: true}, true, "", ""},
		{"below the floor", Admission{Free: 8 * gib, FreeKnown: true, Kept: 100 * gib}, false, ReasonFreeSpace,
			"free space 8.0 GiB is below min_free_gb 10"},
		{"the floor comes before the share", Admission{Free: 9 * gib, FreeKnown: true, Kept: 1000 * gib}, false,
			ReasonFreeSpace, "free space 9.0 GiB is below min_free_gb 10"},
		{"the share is reached", Admission{Free: 160 * gib, FreeKnown: true, Kept: 40 * gib}, false, ReasonQuota,
			"kept checkpoints already use 40.0 GiB, capacity_percent 20 of kept plus free space"},
		{"under the share", Admission{Free: 160 * gib, FreeKnown: true, Kept: 39 * gib}, true, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := p.Admit(tc.a)
			if ok != tc.ok || got.Reason != tc.reason || got.Detail != tc.detail {
				t.Fatalf("Admit = %+v %v", got, ok)
			}
			if !ok && got.State != StateRejected {
				t.Fatalf("a refusal before the move is rejected, got %q", got.State)
			}
		})
	}
}

func TestNewID_ShapeAndCollision(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id, err := NewID()
		if err != nil {
			t.Fatal(err)
		}
		if !ValidID(id) {
			t.Fatalf("id %q is not 12 lowercase hex characters", id)
		}
		if seen[id] {
			t.Fatalf("id %q came twice in 1000", id)
		}
		seen[id] = true
	}
	for _, bad := range []string{"", "3f9a1c0b7d2", "3f9a1c0b7d2e0", "3F9A1C0B7D2E", "3f9a1c0b7d2g", "../escaped!"} {
		if ValidID(bad) {
			t.Fatalf("%q passed as an id", bad)
		}
	}
}

func TestPlanSettle(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	kept := func(id string, age time.Duration, bytes, inodes int64) Entry {
		e := Entry{ID: id, State: EntryKept, CapturedAt: at(now.Add(-age)), ExpiresAt: at(now.Add(48*time.Hour - age))}
		if bytes >= 0 {
			e.Bytes, e.Inodes = i64(bytes), i64(inodes)
		}
		return e
	}
	p := Policy{CapacityPercent: 20, MaxBytes: 32 * gib}
	cases := []struct {
		name    string
		entries []Entry
		fs      Filesystem
		policy  Policy
		want    SettlePlan
	}{
		{"expired kept and evicted go whole", []Entry{
			{ID: "a", State: EntryKept, ExpiresAt: at(now)},
			{ID: "b", State: EntryEvicted, ExpiresAt: at(now.Add(-time.Minute))},
			{ID: "c", State: EntryReserved},
		}, Filesystem{Known: true, Free: 100 * gib}, p, SettlePlan{Expire: []string{"a", "b"}}},
		{"larger than max_gb", []Entry{kept("big", time.Hour, 33*gib, 10)},
			Filesystem{Known: true, Free: 1000 * gib}, p, SettlePlan{Evict: []Eviction{{"big", EvictLargerThanMaxGB}}}},
		{"over the share evicts the oldest first", []Entry{
			kept("new", time.Minute, 10*gib, 10), kept("old", 3*time.Hour, 10*gib, 10), kept("mid", time.Hour, 10*gib, 10),
		}, Filesystem{Known: true, Free: 70 * gib}, p, SettlePlan{Evict: []Eviction{{"old", EvictOverCapacity}}}},
		{"the new one is not spared", []Entry{
			kept("old", time.Hour, 1*gib, 1), kept("new", time.Minute, 30*gib, 1),
		}, Filesystem{Known: true, Free: 50 * gib}, p, SettlePlan{Evict: []Eviction{
			{"old", EvictOverCapacity}, {"new", EvictOverCapacity}}}},
		{"unmeasured entries are neither counted nor evicted", []Entry{
			kept("u", 5*time.Hour, -1, 0), kept("m", time.Hour, 5*gib, 1),
		}, Filesystem{Known: true, Free: 100 * gib}, p, SettlePlan{}},
		{"inodes use the same share by default", []Entry{
			kept("old", time.Hour, gib, 3000), kept("new", time.Minute, gib, 3000),
		}, Filesystem{Known: true, Free: 1000 * gib, FilesTotal: 100000, FilesFree: 20000}, p,
			SettlePlan{Evict: []Eviction{{"old", EvictOverTotalInodes}}}},
		{"a written inode limit wins", []Entry{
			kept("old", time.Hour, gib, 3000), kept("new", time.Minute, gib, 3000),
		}, Filesystem{Known: true, Free: 1000 * gib, FilesTotal: 100000, FilesFree: 20000},
			Policy{CapacityPercent: 20, MaxBytes: 32 * gib, MaxTotalInodes: 1000000}, SettlePlan{}},
		{"no inode count means no inode limit", []Entry{
			kept("old", time.Hour, gib, 3000), kept("new", time.Minute, gib, 3000),
		}, Filesystem{Known: true, Free: 1000 * gib}, p, SettlePlan{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.policy.PlanSettle(tc.entries, tc.fs, now)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("plan = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// 규칙 9절의 표 — 주인 없는 미완료는 trash 로, 도는 예약과 사람이 둔 것은 그대로.
func TestPlanReconcile(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	id := "3f9a1c0b7d2e"
	cases := []struct {
		name string
		f    Found
		want Reconciliation
	}{
		{"a file such as usage.json", Found{Name: "usage.json"}, Leave},
		{"a folder that is not an id", Found{Name: "notes", Dir: true}, WarnUnknown},
		{"no lock file and young", Found{Name: id, Dir: true, Age: time.Minute}, Leave},
		{"no lock file and old", Found{Name: id, Dir: true, Age: 2 * time.Hour}, TrashIt},
		{"held by a running step", Found{Name: id, Dir: true, LockFile: true,
			Record: &Entry{State: EntryReserved}}, Leave},
		{"an ownerless reservation", Found{Name: id, Dir: true, LockFile: true, LockFree: true,
			Record: &Entry{State: EntryReserved}}, TrashIt},
		{"an empty or unreadable record", Found{Name: id, Dir: true, LockFile: true, LockFree: true}, TrashIt},
		{"expired", Found{Name: id, Dir: true, LockFile: true, LockFree: true,
			Record: &Entry{State: EntryEvicted, ExpiresAt: at(now)}}, TrashIt},
		{"kept with no report", Found{Name: id, Dir: true, LockFile: true, LockFree: true,
			Record: &Entry{State: EntryKept, ExpiresAt: at(now.Add(time.Hour))}}, MarkUnknown},
		{"kept and reported", Found{Name: id, Dir: true, LockFile: true, LockFree: true,
			Record: &Entry{State: EntryKept, Report: ReportDelivered, ExpiresAt: at(now.Add(time.Hour))}}, Leave},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PlanReconcile(tc.f, now); got != tc.want {
				t.Fatalf("PlanReconcile = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPolicyModes(t *testing.T) {
	for _, m := range []Mode{ModeOff, ModeOnFailure, ModeAlways} {
		if !m.Valid() {
			t.Fatalf("%q is not valid", m)
		}
	}
	if Mode("sometimes").Valid() || Mode("").Valid() {
		t.Fatal("an unknown mode passed")
	}
	if !strings.HasSuffix(SpoolIn("/s"), "/s/spool") {
		t.Fatalf("SpoolIn = %q", SpoolIn("/s"))
	}
}
