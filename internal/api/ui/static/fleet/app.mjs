import { DashboardView } from '../shared/fleet/view.mjs';
import { ObservationClient } from '../shared/fleet/client.mjs';
import { readToken, endSession, validToken } from './login.mjs';
const root = document.querySelector('#fleet-app'), status = document.querySelector('#fleet-auth-status');
let view, ticker;
function end(message = '') { clearInterval(ticker); client.stop(); view?.destroy(); root.hidden = true; endSession(message); }
const client = new ObservationClient({ mode: 'fleet', token: readToken(), onUnauthorized: () => end('인증이 만료되었습니다. 토큰을 다시 입력하세요 (401).'), onChange: (resources, now) => {
  if (!view && resources.get('nodes')?.data && resources.get('runs')?.data) {
    view = new DashboardView(root, { mode: 'fleet', identity: '실 함대', onRetry: () => client.refresh(true), onLogout: () => end(), onRunSelection: id => client.selectRun(id) });
    root.hidden = false; document.querySelector('#fleet-auth-panel').hidden = true;
  }
  if (view) view.update(resources, now);
  else status.textContent = [...resources.values()].find(r => r.error)?.error || '접속을 확인하고 있습니다…';
} });
document.querySelector('[data-testid="fleet-auth-retry-button"]').addEventListener('click', () => client.refresh(true));
document.querySelector('[data-testid="fleet-auth-back-button"]').addEventListener('click', () => end());
if (!validToken(readToken())) endSession();
else { client.start(); ticker = setInterval(() => view?.update(client.resources), 1000); }
window.addEventListener('pagehide', () => { clearInterval(ticker); client.stop(); view?.destroy(); });
window.addEventListener('pageshow', e => { if (e.persisted) location.reload(); });
