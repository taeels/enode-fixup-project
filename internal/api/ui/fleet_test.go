package ui_test

import (
	"strings"
	"testing"
)

func TestFleetAndDemoModulesAreServed(t *testing.T) {
	for _, path := range []string{"/ui/fleet/", "/ui/fleet/app.mjs", "/ui/fleet/login.mjs", "/ui/shared/fleet/model.mjs", "/ui/shared/fleet/format.mjs", "/ui/shared/fleet/client.mjs", "/ui/shared/fleet/scene.mjs", "/ui/shared/fleet/view.mjs", "/ui/shared/fleet/fleet.css", "/ui/demo/tour.mjs", "/ui/demo/submission.mjs", "/ui/demo/webcam.mjs", "/ui/demo/settings.json"} {
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
	for _, path := range []string{"/ui/tests/model.test.mjs", "/ui/testdata/obs-contract/nodes-states.json", "/ui/shared/fleet/", "/ui/shared/fleet"} {
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
