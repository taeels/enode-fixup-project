package api

import "net/http"

// history 는 토큰 인증을 거친 운영자의 지난 대화 조회다. 원본 계약이나
// 게시 ticket 대신 기존 게스트 결과와 같은 검증된 필드만 반환한다.
// 게스트 결과 경로는 계속 요청 증명을 요구한다.
func (h *galleryHandler) history(w http.ResponseWriter, r *http.Request) {
	if !h.begin(w) {
		return
	}
	id := r.PathValue("id")
	if !galleryRunID.MatchString(id) {
		fail(w, 404, "gallery request not found")
		return
	}
	run, err := h.server.st.GetRun(r.Context(), id)
	if err != nil {
		fail(w, 404, "gallery request not found")
		return
	}
	if _, err := galleryOriginal(run); err != nil {
		fail(w, 404, "gallery request not found")
		return
	}
	h.writeResult(w, r, run)
}
