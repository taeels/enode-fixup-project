import { DashboardView, button } from '../shared/fleet/view.mjs';
import { ObservationClient } from '../shared/fleet/client.mjs';
import { DemoTour } from './tour.mjs';
import { Submission, SubmissionDialog } from './submission.mjs';
import { WebcamWindow } from './webcam.mjs';
const root = document.querySelector('#demo-app');
function update(resources = client.resources, now = performance.now()) {
  view.update(resources, now);
  const state = submission.state, listed = state.run && resources.get('runs')?.data?.runs.some(r => r.run_id === state.run.run_id);
  view.submissionNotice.hidden = !state.message;
  view.submissionNotice.textContent = state.message + (state.phase === 'accepted' && !listed ? ' · 관측 목록에 반영되는 중입니다.' : '');
}
const client = new ObservationClient({ mode: 'demo', onChange: update });
const view = new DashboardView(root, { mode: 'demo', identity: guest.guestName(), onRetry: () => client.refresh(true), onRunSelection: id => client.selectRun(id) });
view.headerActions.append(document.querySelector('#demo-links'));
let storage; try { storage = sessionStorage; } catch { storage = null; }
const submission = new Submission({ submitter: guest.guestName(), storage, onChange: state => { dialog.update(state); update(); }, onRefresh: () => client.refresh(), onAccepted: run => { dialog.close(); view.selectRun(run.run_id, true); client.refresh(); } });
const dialog = new SubmissionDialog(root, { submission, canOpen: () => !tour.open });
const newTask = button('+ 새 작업', 'demo-new-task-button', () => dialog.show(), 'primary'); view.newTaskSlot.append(newTask);
const webcam = new WebcamWindow(view.panel);
const tour = new DemoTour(root, { targets: [webcam.root, view.panel, view.sidebar, newTask], onStart: () => view.changeScene('fleet'), canStart: () => !dialog.open });
const replay = button('화면 안내', 'demo-tour-replay-button', () => tour.begin()); view.headerActions.append(replay);
update(); client.start(); if (!tour.progress.completed) tour.begin();
const ticker = setInterval(() => update(), 1000);
window.addEventListener('pagehide', () => { clearInterval(ticker); client.stop(); submission.destroy(); dialog.destroy(); webcam.destroy(); tour.destroy(); view.destroy(); });
window.addEventListener('pageshow', e => { if (e.persisted) location.reload(); });
