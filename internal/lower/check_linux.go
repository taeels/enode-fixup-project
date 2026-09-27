package lower

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Check 는 준비도 점검 셋이다 (business-rules.md 12절). 아무것도 만들지 않는다 (Peek).
//
//	binding.scratch_filesystem   scratch 와 워크스페이스의 st_dev 가 같고 마운트도 같다 (답 6)
//	lower.owner_uid              워크스페이스 루트의 주인이 노드 uid 다
//	lower.identity               lower.json 이 New · Match · Reused 다.  lowers 가 "" 면 내지 않는다
//
// scratch 가 아직 없으면 가장 가까운 있는 조상으로 본다 — smoke 가 그 아래에 만든다. lowers 는 부르는 쪽이
// 준다 — 이 패키지는 home 을 모른다 (답 9). 워크스페이스를 못 읽으면 앞의 둘만 어긋남으로 낸다.
func Check(lowers, lowerRoot, scratch string, uid int) []Finding {
	ws, err := ReadRoot(lowerRoot)
	if err != nil {
		return workspaceUnread(uid, err)
	}
	sc, scErr := ReadRoot(existingAncestor(scratch))
	out := []Finding{scratchFinding(sc, scErr, ws), ownerFinding(ws, uid)}
	if lowers != "" {
		out = append(out, identityCheck(lowers, ws))
	}
	return out
}

// existingAncestor 는 p 이거나 p 의 가장 가까운 있는 조상이다. 없음 밖의 오류는 그 자리를 그대로 돌려준다 —
// 읽는 쪽이 그 오류를 말한다.
func existingAncestor(p string) string {
	for {
		_, err := os.Stat(p)
		if err == nil || !errors.Is(err, fs.ErrNotExist) {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return p
		}
		p = parent
	}
}

// identityCheck 는 lower.identity 다. 자리를 Peek 하고 lower.json 과 필요하면 state.json 을 읽는다.
func identityCheck(lowers string, ws Root) Finding {
	dirPath := filepath.Join(lowers, ws.Key.String())
	d, err := Peek(lowers, ws)
	if err != nil {
		return identityFinding(dirPath, VerdictBroken, State{}, err, false)
	}
	if d == nil {
		return identityFinding(dirPath, VerdictNew, State{}, nil, false)
	}
	_, v, st, err := d.inspect()
	return identityFinding(d.Path, v, st, err, len(d.Loose) > 0)
}
