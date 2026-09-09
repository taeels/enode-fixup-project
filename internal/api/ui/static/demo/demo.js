import { DashboardView, button } from '../shared/fleet/view.mjs';
import { ObservationClient } from '../shared/fleet/client.mjs';
import { formatRunId } from '../shared/fleet/format.mjs';
import { DemoTour } from './tour.mjs';
import { Submission, SubmissionDialog } from './submission.mjs';
import { GalleryDemo } from './gallery.mjs';
import { GalleryDialog } from './gallery-view.mjs';
import { GalleryHistory } from '../shared/gallery-history.mjs';
import { GalleryHistoryDialog } from '../shared/gallery-history-view.mjs';
import { WebcamWindow } from './webcam.mjs';
const root = document.querySelector('#demo-app');
function update(resources = client.resources, now = performance.now()) {
  view.update(resources, now);
  const state = submission.state, listed = state.run && resources.get('runs')?.data?.runs.some(r => r.run_id === state.run.run_id);
  // 접수 응답은 과거의 상태다. 목록에 나타난 뒤에는 현재 관측 상태만 안내한다.
  view.submissionNotice.hidden = !state.message || (state.phase === 'accepted' && listed);
  view.submissionNotice.textContent = state.phase === 'accepted' ? `${formatRunId(state.run.run_id)} · 접수 확인됨. 관측 목록에 반영되는 중입니다.` : state.message;
  view.submissionNotice.title = state.run?.run_id || '';
}
const client = new ObservationClient({ mode: 'demo', onChange: update });
const view = new DashboardView(root, { mode: 'demo', identity: guest.guestName(), onRetry: () => client.refresh(true), onRunSelection: id => client.selectRun(id), onRunHistory: (id, source) => historyDialog.show(id, source) });
view.headerActions.append(document.querySelector('#demo-links'));
let storage; try { storage = sessionStorage; } catch { storage = null; }
let historyStorage; try { historyStorage = localStorage; } catch { historyStorage = null; }
const history = new GalleryHistory({ storage: historyStorage, onChange: () => historyDialog.render() });
const historyDialog = new GalleryHistoryDialog(root, { history });
const submission = new Submission({ submitter: guest.guestName(), storage, onChange: state => { dialog.update(state); update(); }, onRefresh: () => client.refresh(), onAccepted: run => { dialog.close(); view.selectRun(run.run_id, true); client.refresh(); } });
const gallery = new GalleryDemo({ submitter: guest.guestName(), storage, history, onChange: () => galleryDialog.render(), onAccepted: run => { view.selectRun(run.run_id, true); client.refresh(); } });
const galleryDialog = new GalleryDialog(root, { gallery, returnFocus: () => newTask });
const dialog = new SubmissionDialog(root, { submission, canOpen: () => !tour.open && !galleryDialog.open && !historyDialog.open, onGallery: () => galleryDialog.show() });
const newTask = button('+ 새 작업', 'demo-new-task-button', () => dialog.show(), 'primary'); view.newTaskSlot.append(newTask);
const webcamToggle = button('웹캠', 'demo-webcam-toggle-button', () => webcam.setVisible(webcam.root.hidden)); webcamToggle.setAttribute('aria-expanded', 'true');
const webcam = new WebcamWindow(view.panel, { onVisibilityChange: visible => { webcamToggle.setAttribute('aria-expanded', String(visible)); if (!visible) webcamToggle.focus({ preventScroll: true }); } });
webcam.root.id = 'demo-webcam'; webcamToggle.setAttribute('aria-controls', webcam.root.id); view.headerActions.append(webcamToggle);
const tour = new DemoTour(root, { targets: [webcam.root, view.panel, view.sidebar, newTask], onStart: () => { webcam.setVisible(true); view.changeScene('fleet'); }, canStart: () => !dialog.open && !galleryDialog.open && !historyDialog.open });
const replay = button('화면 안내', 'demo-tour-replay-button', () => tour.begin()); view.headerActions.append(replay);
update(); client.start(); if (!tour.progress.completed) tour.begin();
const ticker = setInterval(() => update(), 1000);
window.addEventListener('pagehide', () => { clearInterval(ticker); client.stop(); submission.destroy(); dialog.destroy(); gallery.destroy(); galleryDialog.destroy(); historyDialog.destroy(); webcam.destroy(); tour.destroy(); view.destroy(); });
window.addEventListener('pageshow', e => { if (e.persisted) location.reload(); });
