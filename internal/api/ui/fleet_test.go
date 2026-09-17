package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/transcriptui"
)

func TestFleetAndDemoModulesAreServed(t *testing.T) {
	for _, path := range []string{"/ui/fleet/", "/ui/fleet/app.mjs", "/ui/fleet/login.mjs", "/ui/shared/fleet/model.mjs", "/ui/shared/fleet/format.mjs", "/ui/shared/fleet/client.mjs", "/ui/shared/fleet/scene.mjs", "/ui/shared/fleet/view.mjs", "/ui/shared/fleet/fleet.css", "/ui/shared/fleet/transcript-poller.mjs", "/ui/shared/transcriptui/card.mjs", "/ui/demo/tour.mjs", "/ui/demo/submission.mjs", "/ui/demo/webcam.mjs", "/ui/demo/settings.json"} {
		r := get(t, path)
		if r.Code != 200 {
			t.Errorf("GET %s = %d", path, r.Code)
		}
		if strings.HasSuffix(path, ".mjs") && !strings.Contains(r.Header().Get("Content-Type"), "javascript") {
			t.Errorf("module MIME for %s = %q", path, r.Header().Get("Content-Type"))
		}
	}
}

func TestObservationTestDataAndModuleDirectoriesAreNotPublic(t *testing.T) {
	for _, path := range []string{"/ui/tests/model.test.mjs", "/ui/testdata/obs-contract/nodes-states.json", "/ui/shared/fleet/", "/ui/shared/fleet", "/ui/shared/transcriptui/", "/ui/shared/transcriptui"} {
		if r := get(t, path); r.Code != 404 {
			t.Errorf("private path %s returned %d", path, r.Code)
		}
	}
}

func TestLandingKeepsGuestEntryAndMasksToken(t *testing.T) {
	body := get(t, "/ui/").Body.String()
	for _, value := range []string{`type="password"`, `data-testid="landing-admin-login-button"`, `data-testid="landing-guest-login-button"`, `/ui/landing.js`, `/ui/shared/guest.js`, `id="admin-login" hidden`} {
		if !strings.Contains(body, value) {
			t.Errorf("landing is missing %s", value)
		}
	}
}

// 카드 렌더러는 이 패키지의 static 트리 밖에서 온다 — 제어판이 같은 바이트를
// 자기 라우트로 내야 해서 둘 다 임포트할 수 있는 잎에 산다. 그 바이트가
// 실제로 같은 출처인지를 여기서 고정한다.
func TestCardModuleIsServedFromTheSharedPackage(t *testing.T) {
	want, err := transcriptui.Files.ReadFile("card.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if got := get(t, "/ui/shared/transcriptui/card.mjs").Body.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("served card module differs from the embedded one (%d vs %d bytes)", len(got), len(want))
	}
}
