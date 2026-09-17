import { DashboardView } from '../shared/fleet/view.mjs';
import { ObservationClient } from '../shared/fleet/client.mjs';
import { TranscriptPoller } from '../shared/fleet/transcript-poller.mjs';
import { readToken, endSession, validToken } from './login.mjs';
import { GalleryHistory } from '../shared/gallery-history.mjs';
import { GalleryHistoryDialog } from '../shared/gallery-history-view.mjs';
// 카드 렌더러는 URL 로 받는다. 상대 경로로 못 적는 이유 - 이 파일은
// internal/transcriptui 가 //go:embed 로 들고 ui.go 가 그 URL 로 내므로
// static 트리의 디스크 경로에 없다. 상대 임포트를 적으면 브라우저에서만
// 서고 node --test 에서는 터진다 (view.mjs 를 전이로 임포트하는 시험 넷이
// 그때 통째로 빨개진다).
const card = await import('/ui/shared/transcriptui/card.mjs');
const root = document.querySelector('#fleet-app'), status = document.querySelector('#fleet-auth-status');
let view, ticker, historyDialog;
function end(message = '') { clearInterval(ticker); client.stop(); transcripts.stop(); historyDialog?.destroy(); view?.destroy(); root.hidden = true; endSession(message); }
const client = new ObservationClient({ mode: 'fleet', token: readToken(), onUnauthorized: () => end('인증이 만료되었습니다. 토큰을 다시 입력하세요 (401).'), onChange: (resources, now) => {
  if (!view && resources.get('nodes')?.data && resources.get('runs')?.data) {
    view = new DashboardView(root, { mode: 'fleet', identity: '실 함대', onRetry: () => client.refresh(true), onLogout: () => end(), onRunSelection: id => client.selectRun(id), onRunHistory: (id, source) => historyDialog.show(id, source), transcripts, card });
    // 카드 타이머는 여기서 선다. 목록의 5초와 별개다 - 목록 폴링을 멈춰도
    // 카드가 자라는 것이 CB4 가 재는 것이고, 그 둘이 한 타이머면 그 조각을
    // 못 잰다.
    transcripts.start();
    const history = new GalleryHistory({ token: readToken(), onChange: () => historyDialog.render() });
    historyDialog = new GalleryHistoryDialog(root, { history });
    root.hidden = false; document.querySelector('#fleet-auth-panel').hidden = true;
  }
  if (view) view.update(resources, now);
  else status.textContent = [...resources.values()].find(r => r.error)?.error || '접속을 확인하고 있습니다…';
} });
const transcripts = new TranscriptPoller({ render: card, token: readToken(), onChange: () => view?.update(client.resources) });
document.querySelector('[data-testid="fleet-auth-retry-button"]').addEventListener('click', () => client.refresh(true));
document.querySelector('[data-testid="fleet-auth-back-button"]').addEventListener('click', () => end());
if (!validToken(readToken())) endSession();
else { client.start(); ticker = setInterval(() => view?.update(client.resources), 1000); }
window.addEventListener('pagehide', () => { clearInterval(ticker); client.stop(); transcripts.stop(); historyDialog?.destroy(); view?.destroy(); });
window.addEventListener('pageshow', e => { if (e.persisted) location.reload(); });
