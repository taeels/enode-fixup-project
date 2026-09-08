import { ObservationClient } from '../shared/fleet/client.mjs';
export const TOKEN_KEY = 'enode.fleet.token';
const ERROR_KEY = 'enode.fleet.login-error';
export const validToken = token => typeof token === 'string' && token.trim().length > 0 && !/[\r\n]/.test(token);
export function readToken() { try { return sessionStorage.getItem(TOKEN_KEY) || ''; } catch { return ''; } }
export function endSession(message = '') {
  try { sessionStorage.removeItem(TOKEN_KEY); if (message) sessionStorage.setItem(ERROR_KEY, message); else sessionStorage.removeItem(ERROR_KEY); } catch { /* 세션 저장소가 없으면 인증을 다시 받는다. */ }
  location.replace('/ui/#fleet');
}
const form = typeof document === 'undefined' ? null : document.querySelector('[data-testid="landing-admin-form"]');
if (form) {
  const admin = document.querySelector('#admin-login'), demo = document.querySelector('#guest-entry'), input = form.querySelector('input');
  const submit = form.querySelector('[type=submit]'), retry = form.querySelector('[type=button]'), status = form.querySelector('[role=status]');
  document.querySelector('#landing-guest-name').textContent = window.guest.guestName();
  let client = null, generation = 0;
  function modeChanged() {
    generation++; client?.stop(); client = null; submit.disabled = false; retry.hidden = true; input.value = '';
    admin.hidden = location.hash !== '#fleet'; demo.hidden = !admin.hidden;
    if (!admin.hidden) input.focus();
  }
  window.addEventListener('hashchange', modeChanged); modeChanged();
  try { status.textContent = sessionStorage.getItem(ERROR_KEY) || ''; sessionStorage.removeItem(ERROR_KEY); } catch { /* 오류 안내를 저장하지 못해도 폼을 제공한다. */ }
  async function login(event) {
    event.preventDefault(); if (submit.disabled) return;
    const token = input.value; if (!validToken(token)) { status.textContent = '빈 토큰이나 줄바꿈이 포함된 토큰은 사용할 수 없습니다.'; input.focus(); return; }
    const attempt = ++generation; client?.stop(); submit.disabled = true; retry.hidden = true; status.textContent = '접속을 확인하고 있습니다…';
    try { sessionStorage.removeItem(TOKEN_KEY); } catch { /* 이전 자료는 앱에 전달하지 않는다. */ }
    client = new ObservationClient({ mode: 'fleet', token, onUnauthorized: () => { if (attempt === generation) { input.value = ''; status.textContent = '인증에 실패했습니다. 토큰을 다시 입력하세요 (401).'; } } });
    await Promise.allSettled(['nodes', 'runs'].map(key => client.request(key)));
    if (attempt !== generation) return;
    const accepted = client.resources.get('nodes')?.data && client.resources.get('runs')?.data;
    if (accepted) {
      try { sessionStorage.setItem(TOKEN_KEY, token); if (readToken() !== token) throw new Error('Unavailable session storage'); }
      catch { status.textContent = '이 탭의 세션 저장소를 사용할 수 없습니다. 브라우저 설정을 확인하세요.'; client.stop(); submit.disabled = false; return; }
      input.value = ''; client.stop(); location.assign('/ui/fleet/');
    } else {
      if (!client.stopped) { status.textContent = [...client.resources.values()].find(r => r.error)?.error || '관측 응답을 확인하지 못했습니다.'; retry.hidden = false; }
      client.stop(); submit.disabled = false;
    }
  }
  form.addEventListener('submit', login); retry.addEventListener('click', login);
  input.addEventListener('paste', e => { if (/[\r\n]/.test(e.clipboardData?.getData('text') || '')) { e.preventDefault(); status.textContent = '줄바꿈이 포함된 토큰은 사용할 수 없습니다.'; } });
  window.addEventListener('pagehide', () => { generation++; client?.stop(); });
}
