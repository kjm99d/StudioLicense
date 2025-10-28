export const state = {
  token: localStorage.getItem('token'),
  currentPage: 1,
  currentStatus: '',
  currentSearch: '',
  currentRole: null,
  permissions: [],
  permissionCatalog: [],
  // global overlay is 13000; keep modal stack above this
  topZIndex: 13000,
};

export function setToken(t) {
  state.token = t;
  if (t) localStorage.setItem('token', t);
  else localStorage.removeItem('token');
}

export function setPermissions(perms) {
  if (Array.isArray(perms)) {
    const unique = Array.from(new Set(perms.filter(Boolean)));
    state.permissions = unique;
  } else {
    state.permissions = [];
  }
}

export function setPermissionCatalog(catalog) {
  state.permissionCatalog = Array.isArray(catalog) ? catalog : [];
}

export function hasPermission(permission) {
  if (!permission) return true;
  if (state.currentRole === 'super_admin') return true;
  return state.permissions.includes(permission);
}

export function clearAuth() {
  state.token = null;
  state.currentRole = null;
  state.permissions = [];
  state.permissionCatalog = [];
  localStorage.removeItem('token');
  localStorage.removeItem('username');
}
