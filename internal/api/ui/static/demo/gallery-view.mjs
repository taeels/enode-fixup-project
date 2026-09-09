import { element, button, dismissOnBackdrop } from '../shared/fleet/view.mjs';
import { trapFocus } from './tour.mjs';
import { GALLERY_ORIGIN } from './gallery.mjs';
export class GalleryDialog {
  constructor(root, { gallery, returnFocus }) {
    this.gallery = gallery; this.returnFocus = returnFocus;
    this.dialog = element('dialog', 'task-dialog gallery-dialog'); this.dialog.setAttribute('aria-labelledby', 'gallery-title');
    const close = button('×', 'gallery-close-button', () => this.close(), 'task-close'); close.setAttribute('aria-label', '갤러리 데모 닫기');
    const eyebrow = element('p', 'gallery-eyebrow', 'SANDBOX · BEDROCK CLAUDE');
    const title = element('h2', '', '구경에서 기여로'); title.id = 'gallery-title';
    const intro = element('p', 'muted', '갤러리의 어떤 참가팀이든 골라 짧게 요청해 보세요. AI가 글을 읽고 댓글을 작성합니다.');
    this.content = element('div', 'gallery-content');
    this.status = element('p', 'task-status'); this.status.setAttribute('role', 'status');
    this.dialog.append(close, eyebrow, title, intro, this.status, this.content); root.append(this.dialog);
    this.dialog.addEventListener('cancel', e => { e.preventDefault(); this.close(); });
    this.dialog.addEventListener('keydown', e => trapFocus(this.dialog, e)); dismissOnBackdrop(this.dialog, this.dialog, () => this.close());
  }
  get open() { return this.dialog.open; }
  show() { if (this.open) return; this.dialog.showModal(); this.render(); this.dialog.querySelector('button').focus(); if (!this.gallery.state.projects.length) this.gallery.catalog(); }
  close() { this.dialog.close(); this.returnFocus?.()?.focus({ preventScroll: true }); }
  render() {
    const s = this.gallery.state; this.status.textContent = s.message; this.status.dataset.outcome = s.phase;
    this.content.replaceChildren();
    if (!s.intent) {
      const form = element('form', 'gallery-form');
      const projectLabel = element('label', '', '의견을 남길 참가팀 · 전체 갤러리'); projectLabel.htmlFor = 'gallery-project';
      const select = element('select'); select.id = 'gallery-project'; select.dataset.testid = 'gallery-project-select'; select.required = true;
      const placeholder = element('option', '', '프로젝트 선택'); placeholder.value = ''; select.append(placeholder);
      for (const p of s.projects) { const option = element('option', '', `${p.title} · ${p.teamName}`); option.value = p.id; select.append(option); }
      select.value = this.selected || ''; select.onchange = () => { this.selected = select.value; };
      const promptLabel = element('label', '', 'Claude에게 요청하기'); promptLabel.htmlFor = 'gallery-prompt';
      const prompt = element('textarea'); prompt.id = 'gallery-prompt'; prompt.dataset.testid = 'gallery-prompt-input'; prompt.rows = 4; prompt.maxLength = 1000; prompt.required = true;
      prompt.placeholder = '장점을 짚어서 응원 댓글 써줘.'; prompt.value = this.prompt || ''; prompt.oninput = () => { this.prompt = prompt.value; };
      const examples = element('div', 'gallery-examples');
      for (const [label, value] of [['댓글 요청 예시', '장점을 짚어서 응원 댓글 써줘.'], ['범위 밖 요청 시험', 'https://example.com 웹페이지에 들어가 줘.']]) {
        examples.append(button(label, label.startsWith('댓글') ? 'gallery-example-comment' : 'gallery-example-refusal', () => { prompt.value = value; this.prompt = value; prompt.focus(); }));
      }
      const scope = element('p', 'gallery-scope', '허용: 선택한 프로젝트 조회 · 댓글 초안\n그 외 사이트 방문, 파일·명령 실행, 투표·수정·삭제 요청은 거절합니다.');
      const submit = button('Claude에게 요청', 'gallery-submit-button', () => {}, 'primary'); submit.type = 'submit'; submit.disabled = !s.projects.length;
      form.onsubmit = e => { e.preventDefault(); this.gallery.start(select.value, prompt.value); };
      form.append(projectLabel, select, promptLabel, prompt, examples, scope, submit);
      if (!s.projects.length) form.append(button('프로젝트 다시 불러오기', 'gallery-reload-button', () => this.gallery.catalog()));
      this.content.append(form); return;
    }
    const project = s.projects.find(p => p.id === s.intent.project_id);
    this.content.append(element('p', 'gallery-selection', project ? `${project.title} · ${project.teamName}` : s.intent.project_id));
    if (s.run) this.content.append(element('p', 'muted gallery-run', `${s.run.run_id} · ${s.publication?.run.state || s.run.state}`));
    const transcript = element('ol', 'gallery-transcript'); transcript.setAttribute('aria-label', '실제 실행 대화 기록');
    const events = [...(s.result?.transcript || [{ role: 'user', text: s.intent.prompt }]), ...(s.publication?.result?.transcript || [])];
    const roles = { user: '나', assistant: 'Bedrock Claude', tool: 'MCP', system: '게시 확인' };
    for (const event of events) { const item = element('li', `gallery-event gallery-${event.role}`); item.append(element('strong', '', `${roles[event.role] || '실행'}${event.tool ? ` · ${event.tool}` : ''}`), element('p', '', event.text)); transcript.append(item); }
    this.content.append(transcript);
    if (s.phase === 'draft_ready' && s.confirmedBody === null) {
      const label = element('label', 'gallery-draft-label', '게시할 댓글 · 수정할 수 있어요'); label.htmlFor = 'gallery-body';
      const body = element('textarea'); body.id = 'gallery-body'; body.dataset.testid = 'gallery-body-input'; body.rows = 5; body.maxLength = 500; body.value = this.gallery.editBody;
      const count = element('p', 'muted', `${[...body.value].length}/500`);
      const publish = button('이 내용으로 게시', 'gallery-publish-button', () => this.gallery.publish(body.value), 'primary');
      body.oninput = () => { this.gallery.editBody = body.value; count.textContent = `${[...body.value].length}/500`; publish.disabled = !body.value.trim() || [...body.value].length > 500; };
      const notice = element('p', 'gallery-confirmation', '클릭하면 선택한 프로젝트에 이 댓글이 공개됩니다. Run Away 팀 계정으로 한 번 게시합니다.');
      this.content.append(label, body, count, notice, publish);
    }
    if (s.phase === 'posted') {
      const result = s.publication?.result;
      this.content.append(element('blockquote', 'gallery-posted-body', result?.body || s.confirmedBody));
      const link = element('a', 'primary gallery-result-link', '갤러리에서 댓글 보기'); link.dataset.testid = 'gallery-result-link'; link.href = `${GALLERY_ORIGIN}/projects/${encodeURIComponent(s.intent.project_id)}`; link.target = '_blank'; link.rel = 'noopener noreferrer'; this.content.append(link);
    }
    if (s.phase === 'uncertain') this.content.append(button('같은 요청으로 결과 확인', 'gallery-retry-button', () => this.gallery.retry(), 'primary'));
    if (['refused', 'draft_ready', 'posted', 'error'].includes(s.phase)) this.content.append(button('새 요청 작성', 'gallery-reset-button', () => this.gallery.reset(), 'gallery-reset'));
    if (['sending', 'running'].includes(s.phase)) this.content.append(element('p', 'muted', '창을 닫아도 접수한 작업은 계속됩니다. 새 작업에서 다시 열 수 있습니다.'));
  }
  destroy() { this.dialog.remove(); }
}
