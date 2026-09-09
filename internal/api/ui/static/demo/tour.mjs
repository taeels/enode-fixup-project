import { element, button, dismissOnBackdrop } from '../shared/fleet/view.mjs';
export const TOUR_KEY = 'enode.demo.tour.v1';
export class TourProgress {
  constructor(storage) { this.storage = storage; this.index = 0; this.completed = false; try { this.completed = storage?.getItem(TOUR_KEY) === '1'; } catch { /* 저장 불가여도 현재 화면의 안내는 가능하다. */ } }
  restart() { this.index = 0; }
  next() { if (this.index < 3) { this.index++; return false; } this.finish(); return true; }
  finish() { this.completed = true; try { this.storage?.setItem(TOUR_KEY, '1'); } catch { /* 다음 방문에 안내가 다시 나올 수 있다. */ } }
}
export function trapFocus(dialog, event) {
  if (event.key !== 'Tab') return;
  const items = [...dialog.querySelectorAll('button, a[href], input, textarea, select, [tabindex="0"]')].filter(el => !el.disabled && !el.hidden && el.getClientRects().length);
  if (!items.length) { event.preventDefault(); dialog.focus(); return; }
  const first = items[0], last = items.at(-1);
  if (event.shiftKey && (document.activeElement === first || !dialog.contains(document.activeElement))) { event.preventDefault(); last.focus(); }
  else if (!event.shiftKey && (document.activeElement === last || !dialog.contains(document.activeElement))) { event.preventDefault(); first.focus(); }
}
export class DemoTour {
  constructor(root, { targets, onStart, canStart = () => true, storage }) {
    this.targets = targets; this.onStart = onStart; this.canStart = canStart;
    if (storage === undefined) { try { storage = localStorage; } catch { storage = null; } }
    this.progress = new TourProgress(storage);
    this.dialog = element('dialog', 'tour-dialog'); this.dialog.setAttribute('aria-label', '데모 화면 안내');
    this.spotlight = element('div', 'tour-spotlight'); this.spotlight.setAttribute('aria-hidden', 'true');
    this.bubble = element('div', 'tour-bubble'); this.count = element('p', 'tour-count'); this.title = element('h2'); this.copy = element('p', 'muted');
    this.next = button('다음', 'demo-tour-next-button', () => { if (this.progress.next()) this.close(); else this.showStep(); }, 'primary');
    this.skip = button('건너뛰기', 'demo-tour-skip-button', () => this.finish());
    this.startButton = button('시작하기', 'demo-tour-start-button', () => this.finish(), 'primary');
    const actions = element('div', 'dialog-actions'); actions.append(this.skip, this.next, this.startButton);
    this.bubble.append(this.count, this.title, this.copy, actions); this.dialog.append(this.spotlight, this.bubble); root.append(this.dialog);
    this.dialog.addEventListener('cancel', e => { e.preventDefault(); this.finish(); });
    this.dialog.addEventListener('keydown', e => trapFocus(this.dialog, e));
    dismissOnBackdrop(this.dialog, this.bubble, () => this.finish());
    this.reposition = () => this.position(); window.addEventListener('resize', this.reposition); window.addEventListener('scroll', this.reposition, true);
  }
  get open() { return this.dialog.open; }
  begin() { if (!this.canStart() || this.open) return; this.returnFocus = document.activeElement; this.onStart(); this.progress.restart(); this.dialog.showModal(); this.showStep(); }
  showStep() {
    const i = this.progress.index, texts = [
      ['실제 장면을 함께 보세요', '방송이 연결되면 이곳에서 LED와 음원 시나리오가 실행되는 장면을 확인할 수 있어요.'],
      ['함대의 상태를 살펴보세요', '모형을 누르면 노드가 광고한 능력과 현재 임대를 볼 수 있어요. 2D와 3D를 전환해 보세요.'],
      ['작업의 흐름을 따라가세요', 'Run을 선택하면 단계와 의존 관계가 열려요. 대기 중인 작업은 목록에 남아 있어요.'],
      ['새 작업을 요청해 보세요', 'LED Toggle 또는 사운드 재생을 선택해 요청할 수 있어요. 접수 결과는 작업 목록과 함께 알려드려요.']
    ];
    this.count.textContent = `${i + 1} / 4`; this.title.textContent = texts[i][0]; this.copy.textContent = texts[i][1];
    this.next.hidden = i === 3; this.startButton.hidden = i !== 3;
    this.targets[i].scrollIntoView({ block: 'nearest', behavior: 'instant' }); this.position();
    (i === 3 ? this.startButton : this.next).focus({ preventScroll: true });
  }
  position() {
    if (!this.open) return;
    const r = this.targets[this.progress.index].getBoundingClientRect(), w = window.innerWidth, h = window.innerHeight;
    Object.assign(this.spotlight.style, { left: `${Math.max(4, r.left - 4)}px`, top: `${Math.max(4, r.top - 4)}px`, width: `${Math.max(0, Math.min(r.width + 8, w - 8))}px`, height: `${Math.max(0, Math.min(r.height + 8, h - 8))}px` });
    const bw = Math.min(380, w - 32); this.bubble.style.width = `${bw}px`;
    const bh = this.bubble.offsetHeight;
    this.bubble.style.left = `${Math.max(16, Math.min(w - bw - 16, r.right + 18 < w - bw ? r.right + 18 : r.left - bw - 18))}px`;
    this.bubble.style.top = `${Math.max(16, Math.min(h - bh - 16, r.top + r.height / 2 - bh / 2))}px`;
  }
  finish() { this.progress.finish(); this.close(); }
  close() { this.dialog.close(); if (this.returnFocus?.isConnected) this.returnFocus.focus({ preventScroll: true }); }
  destroy() { window.removeEventListener('resize', this.reposition); window.removeEventListener('scroll', this.reposition, true); this.dialog.remove(); }
}
