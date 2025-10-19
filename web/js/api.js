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
  
  // 토큰이 없고 headers에 없으면 자동으로 추가
  if (!options.headers) {
    options.headers = {};
  }
  
  const token = localStorage.getItem('token');
  if (token && !options.headers['Authorization']) {
    options.headers['Authorization'] = `Bearer ${token}`;
  }
  
  try {
    if (useGlobal) showLoading();
    const res = await fetch(url, options);
    return res;
  } finally {
    if (useGlobal) hideLoading();
  }
}

export async function getAPI(url) {
  const token = localStorage.getItem('token');
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return apiFetch(url, { method: 'GET', headers });
}

export async function postAPI(url, data) {
  const token = localStorage.getItem('token');
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return apiFetch(url, { method: 'POST', headers, body: JSON.stringify(data) });
}

export async function putAPI(url, data) {
  const token = localStorage.getItem('token');
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return apiFetch(url, { method: 'PUT', headers, body: JSON.stringify(data) });
}

export async function deleteAPI(url) {
  const token = localStorage.getItem('token');
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return apiFetch(url, { method: 'DELETE', headers });
}
