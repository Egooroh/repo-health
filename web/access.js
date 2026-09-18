'use strict';

const tokenDialog = document.querySelector('#token-dialog');
let expiryTimer;
const accessChannel = typeof BroadcastChannel === 'function' ? new BroadcastChannel('repo-health-access') : null;
if (accessChannel) accessChannel.onmessage = () => { clearPersonalView(); initializeAccess(); };

function accessUI() {
  document.querySelector('#access-status').textContent = state.connected ? 'Токен подключён · сессия на 1 час' : 'Токен не подключён';
  document.querySelector('#connect-token').textContent = state.connected ? 'Заменить токен' : 'Подключить токен';
  document.querySelector('#disconnect-token').hidden = !state.connected;
  document.querySelector('#access-scope option[value="personal"]').disabled = !state.connected;
  modeUI();
}

function clearPersonalView() {
  // Invalidate all in-flight view loads before changing the credential context.
  ++state.serial;
  clearTimeout(state.timer);
  state.items = [];
  main.innerHTML = '';
}

function scheduleExpiry(expiresAt) {
  clearTimeout(expiryTimer);
  if (!expiresAt) return;
  expiryTimer = setTimeout(() => {
    clearPersonalView();
    state.connected = false; state.personal = false;
    accessUI();
    location.hash = '#/repositories';
    load();
    toast('Личная сессия завершена. Подключите токен снова.');
  }, Math.max(0, new Date(expiresAt).getTime() - Date.now()));
}

document.querySelector('#connect-token').addEventListener('click', () => {
  document.querySelector('#token-error').hidden = true;
  tokenDialog.showModal();
  document.querySelector('#sourcecraft-token').focus();
});
document.querySelector('#close-token').addEventListener('click', () => tokenDialog.close());
tokenDialog.addEventListener('close', () => document.querySelector('#token-form').reset());
document.querySelector('#token-form').addEventListener('submit', async event => {
  event.preventDefault();
  const button = document.querySelector('#save-token');
  const input = document.querySelector('#sourcecraft-token');
  const error = document.querySelector('#token-error');
  button.disabled = true; error.hidden = true;
  // The token goes only in this POST body; clear the input immediately.
  const request = api('/session', {token: input.value});
  input.value = '';
  try {
    const result = await request;
    clearPersonalView();
    state.connected = true; state.personal = true; state.demo = false;
    scheduleExpiry(result.expires_at); accessUI(); tokenDialog.close();
    accessChannel?.postMessage('changed');
    location.hash = viewLink('repositories'); await load();
    toast('SourceCraft подключён. Теперь можно запустить личный анализ.');
  } catch (failure) { error.textContent = failure.message; error.hidden = false; }
  finally { button.disabled = false; }
});

document.querySelector('#disconnect-token').addEventListener('click', async event => {
  event.target.disabled = true;
  try {
    await api('/session/disconnect', {});
    clearPersonalView(); scheduleExpiry(null);
    accessChannel?.postMessage('changed');
    state.connected = false; state.personal = false;
    accessUI(); location.hash = '#/repositories'; await load();
    toast('Токен отключён. Личные данные сессии удалены.');
  } catch (error) { toast(error.message); }
  finally { event.target.disabled = false; }
});

document.querySelector('#access-scope').addEventListener('change', async event => {
  clearPersonalView(); state.personal = event.target.value === 'personal';
  state.demo = false; modeUI();
  location.hash = viewLink('repositories'); await load();
});

async function initializeAccess() {
  try {
    const result = await api('/session');
    state.connected = result.connected;
    state.personal = location.hash.startsWith('#/personal/') || (result.connected && !location.hash);
    if (!location.hash && state.personal) location.hash = viewLink('repositories');
    scheduleExpiry(result.expires_at);
  } catch (error) { state.connected = false; state.personal = false; toast(error.message); }
  accessUI(); await load();
}

// Other tabs share the same cookie. Recheck when returning to a tab and remove
// stale rendered data before any further API reads.
window.addEventListener('pageshow', event => { if (event.persisted) { clearPersonalView(); initializeAccess(); } });
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') { clearPersonalView(); initializeAccess(); }
});

initializeAccess();
