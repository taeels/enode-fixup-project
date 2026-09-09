import { element, button, dismissOnBackdrop } from '../shared/fleet/view.mjs';
import { trapFocus } from './tour.mjs';
import { GALLERY_ORIGIN } from './gallery.mjs';
export class GalleryDialog {
  constructor(root, { gallery, returnFocus }) {
    this.gallery = gallery; this.returnFocus = returnFocus;
    this.dialog = element('dialog', 'task-dialog gallery-dialog'); this.dialog.setAttribute('aria-labelledby', 'gallery-title');
    const close = button('×', 'gallery-close-button', () => this.close(), 'task-close'); close.setAttribute('aria-label', '갤러리 데모 닫기');
    const eyebrow = element('p', 'gallery-eyebrow', 'CLAUDE');
    const title = element('h2', '', '해커톤에 의견 남기기'); title.id = 'gallery-title';
    const intro = element('p', 'muted', '어떤 팀에 어떤 말을 전하고 싶으세요?');
    this.content = element('div', 'gallery-content');
    this.status = element('p', 'task-status'); this.status.setAttribute('role', 'status');
    this.dialog.append(close, eyebrow, title, intro, this.status, this.content); root.append(this.dialog);
    this.dialog.addEventListener('cancel', e => { e.preventDefault(); this.close(); });
    this.dialog.addEventListener('keydown', e => trapFocus(this.dialog, e)); dismissOnBackdrop(this.dialog, this.dialog, () => this.close());
  }
  get open() { return this.dialog.open; }
  show() { if (this.open) return; this.dialog.showModal(); this.render(); (this.dialog.querySelector('textarea') || this.dialog.querySelector('button')).focus(); }
  close() { this.dialog.close(); this.returnFocus?.()?.focus({ preventScroll: true }); }
  composer(followup = false) {
    const form = element('form', 'gallery-form');
    const label = element('label', '', '메시지'); label.htmlFor = 'gallery-prompt';
    const prompt = element('textarea'); prompt.id = 'gallery-prompt'; prompt.dataset.testid = 'gallery-prompt-input'; prompt.rows = 6; prompt.maxLength = 1000; prompt.required = true;
    prompt.placeholder = followup ? '팀 이름이나 추가 설명을 적어주세요.' : '외부 Mac mini의 enode 샌드박스가\nDDTHON 페이지에 연결되어 있어요.\nBedrock Claude에게 댓글을 요청해 보세요!\n\n예시) Run Away 팀에 응원 댓글 달아줘'; prompt.value = this.prompt || ''; prompt.oninput = () => { this.prompt = prompt.value; };
    const submit = button('보내기', 'gallery-submit-button', () => {}, 'primary'); submit.type = 'submit';
    form.onsubmit = e => {
      e.preventDefault(); const message = prompt.value;
      if (!message.trim()) return;
      const previous = followup ? this.gallery.state.intent.prompt : '';
      if (followup) this.gallery.reset();
      this.prompt = '';
      this.gallery.start(previous ? `${previous}\n\n추가 설명: ${message}` : message);
    };
    form.append(label, prompt, submit); return form;
  }
  render() {
    const s = this.gallery.state; this.status.textContent = s.message; this.status.dataset.outcome = s.phase;
    this.content.replaceChildren();
    if (!s.intent) { this.content.append(this.composer()); return; }
    const transcript = element('ol', 'gallery-transcript'); transcript.setAttribute('aria-label', '실제 실행 대화 기록');
    const events = [...(s.result?.transcript || [{ role: 'user', text: s.intent.prompt }]), ...(s.publication?.result?.transcript || [])];
    const roles = { user: '나', assistant: 'Claude', tool: '작업', system: '진행 상황' };
    const tools = { list_projects: '참가팀 확인', get_project: '프로젝트 읽기', get_comments: '댓글 읽기', post_comment: '댓글 게시', post_confirmed_comment: '댓글 게시' };
    for (const event of events) { const item = element('li', `gallery-event gallery-${event.role}`); item.append(element('strong', '', event.tool ? tools[event.tool] || roles[event.role] : roles[event.role] || '진행 상황'), element('p', '', event.text)); transcript.append(item); }
    this.content.append(transcript);
    if (s.phase === 'posted') {
      const result = s.publication?.result || s.result;
      const selected = result?.project_id || s.intent.project_id;
      this.content.append(element('blockquote', 'gallery-posted-body', result?.body || s.confirmedBody));
      if (selected && /^[a-zA-Z0-9_-]{1,64}$/.test(selected)) {
        const link = element('a', 'primary gallery-result-link', '댓글 보러 가기'); link.dataset.testid = 'gallery-result-link'; link.href = `${GALLERY_ORIGIN}/projects/${encodeURIComponent(selected)}`; link.target = '_blank'; link.rel = 'noopener noreferrer'; this.content.append(link);
      }
    }
    if (s.phase === 'needs_project') this.content.append(this.composer(true));
    if (s.phase === 'uncertain') this.content.append(button('다시 연결', 'gallery-retry-button', () => this.gallery.retry(), 'primary'));
    if (['refused', 'draft_ready', 'answered', 'posted', 'error'].includes(s.phase)) this.content.append(button('새 메시지', 'gallery-reset-button', () => this.gallery.reset(), 'gallery-reset'));
  }
  destroy() { this.dialog.remove(); }
}
