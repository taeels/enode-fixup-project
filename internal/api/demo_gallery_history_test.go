package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/taeels/enode/internal/store"
)

func TestGalleryHistory_AuthenticatedReadOfExistingRecord(t *testing.T) {
	h := galleryTestHandler(t)
	h.server.cfg.DemoGallery = true
	id := "gallery-" + strings.Repeat("a", 64)
	result := galleryResult{Outcome: "posted", Message: "Published", ProjectID: "project-1", Body: "Well done", URL: "https://outside.test",
		Transcript: []galleryEvent{{Role: "user", Text: "Write a comment"}, {Role: "tool", Tool: "get_comments", Text: "Read comments"}, {Role: "assistant", Text: "Published"}}}
	galleryFixture(t, h, id, result)
	router := h.server.Handler()
	path := "/v1/demo/gallery/history/" + id
	for _, token := range []string{"", "wrong-token", galleryProof} {
		demoTestStatus(t, demoTestCall(router, "GET", path, "", token), 401)
	}
	w := demoTestCall(router, "GET", path, "", h.server.cfg.Token)
	demoTestStatus(t, w, 200)
	if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("history may be cached")
	}
	var response struct {
		Result galleryResult `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Transcript) != 3 || response.Result.URL != galleryOrigin+"/projects/project-1" {
		t.Fatal("saved conversation or fixed-origin link missing")
	}
	for _, secret := range []string{galleryProof, h.server.cfg.Token, "owner_hash", "ticket", galleryWorker, "outside.test"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("private field exposed: %s", secret)
		}
	}
	// 읽기 경로에 POST를 허용하지 않으며 게스트 경로의 증명도 계속 필요하다.
	demoTestStatus(t, demoTestCall(router, "POST", path, "{}", h.server.cfg.Token), http.StatusMethodNotAllowed)
	demoTestStatus(t, demoTestCall(router, "GET", "/v1/demo/gallery/runs/"+id, "", ""), 404)
	w = galleryCall(h.result, "GET", "/v1/demo/gallery/runs/"+id, "", galleryProof)
	demoTestStatus(t, w, 200)
	run, err := h.server.st.GetRun(context.Background(), id)
	if err != nil || run.State != store.StateSucceeded || !h.server.records.Sealed(id) {
		t.Fatal("read changed the sealed run", err)
	}
}

func TestGalleryHistory_LegacyPublicationAndInvalidContract(t *testing.T) {
	h := galleryTestHandler(t)
	id := "gallery-" + strings.Repeat("b", 64)
	galleryFixture(t, h, id, galleryResult{Outcome: "draft_ready", Message: "Draft", Body: "Nice work", Transcript: []galleryEvent{{Role: "assistant", Text: "Draft"}}})
	galleryFixture(t, h, strings.Replace(id, "gallery-", "gallery-post-", 1), galleryResult{Outcome: "posted", Message: "Posted", ProjectID: "project-1", Transcript: []galleryEvent{{Role: "tool", Tool: "post_confirmed_comment", Text: "Posted once"}}})
	w := galleryCall(h.history, "GET", "/v1/demo/gallery/history/"+id, "", "")
	demoTestStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `"publication"`) || !strings.Contains(w.Body.String(), "Posted once") {
		t.Fatal("legacy publication missing")
	}
	forgedID := "gallery-" + strings.Repeat("c", 64)
	c, err := galleryContract(forgedID, galleryTestJob())
	if err != nil {
		t.Fatal(err)
	}
	c.Steps[0].Run[0] = "/bin/sh"
	if err := h.server.st.CreateRun(context.Background(), store.Run{RunID: forgedID, State: store.StateSucceeded, Contract: c}, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, invalidID := range []string{forgedID, "not-gallery", "gallery-" + strings.Repeat("d", 64)} {
		demoTestStatus(t, galleryCall(h.history, "GET", "/v1/demo/gallery/history/"+invalidID, "", ""), 404)
	}
}
