// Editor settings kept only in this browser (localStorage). AppSecret never leaves the browser
// except as part of the request body to our own server (see editor/API.md).

const ACCOUNTS_KEY = 'editor_wechat_accounts';
const AI_KEY = 'editor_ai_key';
const DRAFT_META_KEY = 'editor_draft_meta';

/** → { accounts: [{id, name, appid, secret}], defaultId } */
export function loadAccounts() {
  try {
    const data = JSON.parse(localStorage.getItem(ACCOUNTS_KEY) || 'null');
    if (data && Array.isArray(data.accounts)) {
      const accounts = data.accounts.filter(a => a && a.id && a.appid);
      const defaultId = accounts.some(a => a.id === data.defaultId) ? data.defaultId : (accounts[0]?.id || '');
      return { accounts, defaultId };
    }
  } catch {
    // ignore
  }
  return { accounts: [], defaultId: '' };
}

export function saveAccounts(state) {
  localStorage.setItem(ACCOUNTS_KEY, JSON.stringify({ accounts: state.accounts, defaultId: state.defaultId }));
}

export function newAccountId() {
  return `acc-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

/** → {id, name} | null */
export function loadAiKey() {
  try {
    const k = JSON.parse(localStorage.getItem(AI_KEY) || 'null');
    if (k && k.id) return { id: Number(k.id), name: k.name || '' };
  } catch {
    // ignore
  }
  return null;
}

export function saveAiKey(key) {
  if (key && key.id) localStorage.setItem(AI_KEY, JSON.stringify({ id: key.id, name: key.name || '' }));
  else localStorage.removeItem(AI_KEY);
}

const EMPTY_META = { title: '', titleSource: '', author: '', digest: '', sourceUrl: '', coverImageId: '', openComment: false };

/** Draft fields (title / author / digest …) kept between visits. */
export function loadDraftMeta() {
  try {
    const m = JSON.parse(localStorage.getItem(DRAFT_META_KEY) || 'null');
    if (m && typeof m === 'object') return { ...EMPTY_META, ...m };
  } catch {
    // ignore
  }
  return { ...EMPTY_META };
}

export function saveDraftMeta(meta) {
  try {
    localStorage.setItem(DRAFT_META_KEY, JSON.stringify(meta));
  } catch {
    // ignore
  }
}
