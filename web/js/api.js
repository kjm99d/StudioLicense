import { state } from './state.js';

export const API_BASE_URL = 'http://localhost:8080';

function showLoading() {
  const el = document.getElementById('global-loading');
  if (el) el.style.display = 'flex';
}
function hideLoading() {
  const el = document.getElementById('global-loading');
  if (el) el.style.display = 'none';
}

export async function apiFetch(url, options = {}) {
  const useGlobal = !options._noGlobalLoading;
  try {
    if (useGlobal) showLoading();
    const res = await fetch(url, options);
    return res;
  } finally {
    if (useGlobal) hideLoading();
  }
}
