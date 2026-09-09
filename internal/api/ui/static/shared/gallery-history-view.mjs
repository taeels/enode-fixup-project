import { element, button, dismissOnBackdrop } from './fleet/view.mjs';
import { formatRunId } from './fleet/format.mjs';
import { trapFocus } from '../demo/tour.mjs';
import { galleryTranscript } from '../demo/gallery-view.mjs';
import { GALLERY_ORIGIN } from '../demo/gallery.mjs';

export class GalleryHistoryDialog {
  constructor(root, { history }) {
    this.history = history;
    this.dialog = element('dialog', 'gallery-history-dialog'); this.dialog.setAttribute('aria-labelledby', 'gallery-history-title');
    const title = element('h2', '', '갤러리 대화 기록'); title.id = 'gallery-history-title';
    const heading = element('header', 'gallery-history-heading');
    this.closeButton = button('닫기', 'gallery-history-close-button', () => this.close());
    heading.append(title, this.closeButton);
    this.runLabel = element('p', 'muted gallery-history-run');
    this.status = element('p', 'task-status'); this.status.setAttribute('role', 'status');
    this.content = element('div', 'gallery-content');
    this.retry = button('다시 조회', 'gallery-history-retry-button', () => history.read(history.state.id));
    this.dialog.append(heading, this.runLabel, this.status, this.content, this.retry); root.append(this.dialog);
    this.dialog.addEventListener('cancel', e => { e.preventDefault(); this.close(); });
    this.dialog.addEventListener('keydown', e => trapFocus(this.dialog, e));
    dismissOnBackdrop(this.dialog, this.dialog, () => this.close());
  }
  get open() { return this.dialog.open; }
  show(id, source) {
    this.source = source;
    if (!this.open) this.dialog.showModal();
    this.closeButton.focus(); this.history.read(id);
  }
  close() {
    this.history.cancel(); this.dialog.close();
    this.source?.focus({ preventScroll: true });
  }
  render() {
    const s = this.history.state;
    this.runLabel.textContent = s.id ? formatRunId(s.id) : ''; this.runLabel.title = s.id || '';
    this.status.textContent = s.message || ''; this.status.dataset.outcome = s.phase;
    this.content.replaceChildren(); this.retry.hidden = s.phase === 'loading';
    if (!s.data) return;
    const events = [...(s.data.result?.transcript || []), ...(s.data.publication?.result?.transcript || [])];
    const result = s.data.publication?.result || s.data.result;
    this.status.dataset.outcome = result?.outcome || s.phase;
    if (events.length) this.content.append(galleryTranscript(events, s.data.run.submitter || '요청자'));
    else if (result) this.content.append(element('p', 'muted', '저장된 대화 메시지가 없습니다.'));
    if (result?.body) this.content.append(element('blockquote', 'gallery-posted-body', result.body));
    if (result?.outcome === 'posted' && /^[a-zA-Z0-9_-]{1,64}$/.test(result.project_id || '')) {
      const link = element('a', 'primary gallery-result-link', '댓글 보러 가기'); link.dataset.testid = 'gallery-history-result-link';
      link.href = `${GALLERY_ORIGIN}/projects/${encodeURIComponent(result.project_id)}`; link.target = '_blank'; link.rel = 'noopener noreferrer'; this.content.append(link);
    }
    this.retry.hidden = !!result && (!s.data.publication || !!s.data.publication.result);
  }
  destroy() { this.history.destroy(); this.dialog.remove(); }
}
