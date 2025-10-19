export const state = {
  token: localStorage.getItem('token'),
  currentPage: 1,
  currentStatus: '',
  currentSearch: '',
  currentRole: null,
  // global overlay is 13000; keep modal stack above this
  topZIndex: 13000,
};

export function setToken(t) {
  state.token = t;
  if (t) localStorage.setItem('token', t);
  else localStorage.removeItem('token');
}

export function clearAuth() {
  state.token = null;
  localStorage.removeItem('token');
  localStorage.removeItem('username');
}
