import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { formatDateTime, escapeHtml } from '../utils.js';

let permissionCatalog = [];
let permissionCatalogLoaded = false;
let permissionCatalogPromise = null;
const adminsCache = new Map();
const RESOURCE_TYPES = [
  {
    key: 'licenses',
    label: '라이선스',
    placeholder: '라이선스 키, 고객명 검색',
    summaryLabel: '라이선스',
  },
  {
    key: 'policies',
    label: '정책',
    placeholder: '정책명 검색',
    summaryLabel: '정책',
  },
  {
    key: 'products',
    label: '제품',
    placeholder: '제품명 검색',
    summaryLabel: '제품',
  },
];
const resourceTypeIndex = new Map(RESOURCE_TYPES.map((type) => [type.key, type]));
const resourceCatalogCache = new Map(); // resourceType -> { items, loaded, loading, error }
const resourcePermissionCache = new Map(); // adminId -> { resourceType: { mode, selected: Set, search: '' }, __loaded: boolean }
const RESOURCE_MODES = new Set(['all', 'none', 'own', 'custom']);
const resourceUIState = {
  activeType: RESOURCE_TYPES[0]?.key || null,
  activeAdminId: null,
};

async function ensurePermissionCatalog() {
  if (permissionCatalogLoaded) {
    return permissionCatalog;
  }
  if (permissionCatalogPromise) {
    await permissionCatalogPromise;
    return permissionCatalog;
  }

  permissionCatalogPromise = (async () => {
    try {
      const res = await apiFetch(`${API_BASE_URL}/api/admin/permissions/catalog`, {
        headers: { 'Authorization': `Bearer ${state.token}` },
        _noGlobalLoading: true
      });
      const body = await res.json();
      if (res.ok && body.status === 'success') {
        permissionCatalog = Array.isArray(body.data) ? body.data : [];
        permissionCatalogLoaded = true;
      } else {
        permissionCatalog = [];
        throw new Error(body.message || '권한 목록을 불러오지 못했습니다.');
      }
    } catch (err) {
      console.error('Failed to load permission catalog:', err);
      permissionCatalog = [];
      throw err;
    } finally {
      permissionCatalogPromise = null;
    }
  })();

  try {
    await permissionCatalogPromise;
  } catch (err) {
    // ignore here; callers can decide how to handle missing catalog
  }
  return permissionCatalog;
}

function groupPermissionsByCategory() {
  const map = new Map();
  permissionCatalog.forEach((perm) => {
    const category = perm?.category || '기타';
    if (!map.has(category)) {
      map.set(category, []);
    }
    map.get(category).push(perm);
  });
  return map;
}

function renderPermissionChecklist(container, selectedKeys = []) {
  if (!container) return;

  if (!permissionCatalogLoaded || permissionCatalog.length === 0) {
    container.innerHTML = '<p class="permission-empty">권한 목록을 불러오지 못했습니다.</p>';
    return;
  }

  const selectedSet = new Set(Array.isArray(selectedKeys) ? selectedKeys : []);
  const groups = groupPermissionsByCategory();
  container.innerHTML = '';

  groups.forEach((permissions, category) => {
    const group = document.createElement('div');
    group.className = 'permission-group';

    const toggle = document.createElement('button');
    toggle.type = 'button';
    toggle.className = 'permission-group-toggle';
    toggle.innerHTML = `
      <span class="permission-group-title">${escapeHtml(category)}</span>
      <span class="permission-group-summary"></span>
      <span class="permission-group-icon">▼</span>
    `;
    group.appendChild(toggle);

    const body = document.createElement('div');
    body.className = 'permission-group-body';

    permissions.forEach((perm) => {
      const item = document.createElement('label');
      item.className = 'permission-item';

      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.value = perm.key;
      checkbox.dataset.permissionKey = perm.key;
      if (selectedSet.has(perm.key)) {
        checkbox.checked = true;
        item.classList.add('selected');
      }
      item.appendChild(checkbox);

      const text = document.createElement('div');
      text.className = 'permission-item-text';
      const label = document.createElement('div');
      label.className = 'permission-item-label';
      label.textContent = perm.label || perm.key;
      text.appendChild(label);

      if (perm.description) {
        const desc = document.createElement('div');
        desc.className = 'permission-item-desc';
        desc.textContent = perm.description;
        text.appendChild(desc);
      }

      item.appendChild(text);

      checkbox.addEventListener('change', () => {
        item.classList.toggle('selected', checkbox.checked);
        updatePermissionGroupSummary(group);
      });

      body.appendChild(item);
    });

    toggle.addEventListener('click', () => {
      group.classList.toggle('collapsed');
      updatePermissionGroupSummary(group);
    });

    group.appendChild(body);
    container.appendChild(group);
    updatePermissionGroupSummary(group);
  });
}

function ensureAdminResourceState(adminId) {
  if (!resourcePermissionCache.has(adminId)) {
    const defaults = {};
    RESOURCE_TYPES.forEach(({ key }) => {
      defaults[key] = {
        mode: 'all',
        selected: new Set(),
        search: '',
      };
    });
    defaults.__loaded = false;
    resourcePermissionCache.set(adminId, defaults);
  }
  return resourcePermissionCache.get(adminId);
}

function normalizeResourceMode(mode) {
  const normalized = String(mode || '').trim().toLowerCase();
  return RESOURCE_MODES.has(normalized) ? normalized : 'all';
}

function applyResourcePermissionsToState(adminId, resourcePermissions) {
  const state = ensureAdminResourceState(adminId);
  RESOURCE_TYPES.forEach(({ key }) => {
    const entry = state[key];
    const incoming = resourcePermissions?.[key];
    entry.mode = normalizeResourceMode(incoming?.mode);
    const selectedIds = Array.isArray(incoming?.selected_ids) ? incoming.selected_ids : [];
    entry.selected = new Set(selectedIds);
  });
  state.__loaded = true;
  return state;
}

function serializeResourcePermissions(adminId) {
  const state = ensureAdminResourceState(adminId);
  const payload = {};
  RESOURCE_TYPES.forEach(({ key }) => {
    const entry = state[key];
    payload[key] = {
      mode: normalizeResourceMode(entry.mode),
      selected_ids: Array.from(entry.selected || []),
    };
  });
  return payload;
}

function getAdminResourceModeLabel(mode) {
  switch (mode) {
    case 'all':
      return '모두 허용';
    case 'none':
      return '모두 차단';
    case 'own':
      return '내가 생성한 항목';
    case 'custom':
      return '선택한 항목만';
    default:
      return mode;
  }
}

function getResourceModeChipClass(mode) {
  switch (normalizeResourceMode(mode)) {
    case 'none':
      return 'resource-mode-chip resource-mode-chip--none';
    case 'own':
      return 'resource-mode-chip resource-mode-chip--own';
    case 'custom':
      return 'resource-mode-chip resource-mode-chip--custom';
    case 'all':
    default:
      return 'resource-mode-chip resource-mode-chip--all';
  }
}

function getResourceUIElements() {
  const modal = document.getElementById('manage-admin-permissions-modal');
  if (!modal) return null;
  return {
    pane: modal.querySelector('#resource-permission-pane'),
    typeTabs: modal.querySelector('[data-role="resource-type-tabs"]'),
    modeContainer: modal.querySelector('[data-role="resource-mode"]'),
    helper: modal.querySelector('[data-role="resource-helper"]'),
    searchInput: modal.querySelector('[data-role="resource-search"]'),
    refreshButton: modal.querySelector('[data-role="resource-refresh"]'),
    list: modal.querySelector('[data-role="resource-list"]'),
    summary: modal.querySelector('[data-role="resource-summary"]'),
  };
}

function getResourceCatalog(type) {
  if (!resourceCatalogCache.has(type)) {
    resourceCatalogCache.set(type, {
      items: [],
      loaded: false,
      loading: false,
      error: null,
      lastFetched: 0,
      loadingPromise: null,
    });
  }
  return resourceCatalogCache.get(type);
}

async function loadResourceCatalog(type, forceReload = false) {
  const catalog = getResourceCatalog(type);
  if (catalog.loaded && !forceReload) {
    return catalog;
  }
  if (catalog.loading && catalog.loadingPromise) {
    await catalog.loadingPromise;
    return resourceCatalogCache.get(type);
  }

  const loader = async () => {
    try {
      const items = await fetchResourceItems(type);
      catalog.items = items;
      catalog.loaded = true;
      catalog.error = null;
      catalog.lastFetched = Date.now();
    } catch (err) {
      console.error(`Failed to load ${type} resources:`, err);
      catalog.error = err?.message || '리소스를 불러오지 못했습니다.';
    } finally {
      catalog.loading = false;
      catalog.loadingPromise = null;
    }
    return catalog;
  };

  catalog.loading = true;
  catalog.loadingPromise = loader();
  await catalog.loadingPromise;
  return resourceCatalogCache.get(type);
}

async function fetchResourceItems(type) {
  const config = resourceTypeIndex.get(type);
  if (!config) return [];

  switch (type) {
    case 'licenses': {
      const res = await apiFetch(`${API_BASE_URL}/api/admin/licenses?page=1&page_size=100`, {
        headers: { Authorization: `Bearer ${state.token}` },
        _noGlobalLoading: true,
      });
      const body = await res.json();
      if (res.status === 403) {
        throw new Error(`${config?.label || type} 목록에 접근할 권한이 없습니다. 기능 권한에서 조회 권한을 먼저 부여하세요.`);
      }
      if (res.ok && body.status === 'success') {
        const items = Array.isArray(body.data) ? body.data : [];
        return items.map((license) => ({
          id: license.id,
          name: license.license_key,
          description: `${license.customer_name || '-'} · ${license.product_name || '-'}`,
        }));
      }
      throw new Error(body?.message || '라이선스를 불러오지 못했습니다.');
    }
    case 'policies': {
      const res = await apiFetch(`${API_BASE_URL}/api/admin/policies`, {
        headers: { Authorization: `Bearer ${state.token}` },
        _noGlobalLoading: true,
      });
      const body = await res.json();
      if (res.status === 403) {
        throw new Error(`${config?.label || type} 목록에 접근할 권한이 없습니다. 정책 조회 기능 권한을 부여하세요.`);
      }
      if (res.ok && body.status === 'success') {
        const items = Array.isArray(body.data) ? body.data : [];
        return items.map((policy) => ({
          id: policy.id,
          name: policy.policy_name,
          description: `업데이트: ${formatDateTime(policy.updated_at)}`,
        }));
      }
      throw new Error(body?.message || '정책을 불러오지 못했습니다.');
    }
    case 'products': {
      const res = await apiFetch(`${API_BASE_URL}/api/admin/products`, {
        headers: { Authorization: `Bearer ${state.token}` },
        _noGlobalLoading: true,
      });
      const body = await res.json();
      if (res.status === 403) {
        throw new Error(`${config?.label || type} 목록에 접근할 권한이 없습니다. 제품 조회 기능 권한을 부여하세요.`);
      }
      if (res.ok && body.status === 'success') {
        const items = Array.isArray(body.data) ? body.data : [];
        return items.map((product) => ({
          id: product.id,
          name: product.name,
          description: product.description || '-',
        }));
      }
      throw new Error(body?.message || '제품을 불러오지 못했습니다.');
    }
    default:
      return [];
  }
}

function renderResourceTypeTabs(adminId) {
  const ui = getResourceUIElements();
  if (!ui?.typeTabs) return;
  const state = ensureAdminResourceState(adminId);
  const activeType = resourceUIState.activeType || RESOURCE_TYPES[0]?.key;
  ui.typeTabs.innerHTML = RESOURCE_TYPES.map((type) => {
    const entry = state[type.key] || { mode: 'all', selected: new Set() };
    const config = resourceTypeIndex.get(type.key);
    const chipClass = getResourceModeChipClass(entry.mode);
    const modeLabel = getAdminResourceModeLabel(entry.mode);
    const customDetail =
      entry.mode === 'custom'
        ? entry.selected.size > 0
          ? `선택 ${entry.selected.size}개`
          : '선택된 항목 없음'
        : '';
    return `
      <button type="button" class="resource-type-tab ${activeType === type.key ? 'is-active' : ''}" data-resource-type="${type.key}">
        <span class="resource-type-label">${escapeHtml(config?.label || type.label || type.key)}</span>
        <div class="resource-type-meta">
          <span class="${chipClass}">${escapeHtml(modeLabel)}</span>
          ${customDetail ? `<span class="resource-type-detail">${escapeHtml(customDetail)}</span>` : ''}
        </div>
      </button>
    `;
  }).join('');
}

function updateResourceToolsState(adminId, type) {
  const ui = getResourceUIElements();
  if (!ui) return;
  const state = ensureAdminResourceState(adminId)[type];
  if (!state) return;
  const config = resourceTypeIndex.get(type);
  const isCustom = state.mode === 'custom';
  if (ui.searchInput) {
    const placeholder = config?.placeholder || '검색어를 입력하세요';
    ui.searchInput.placeholder = isCustom ? placeholder : '선택한 항목만 모드에서 사용할 수 있습니다';
    const currentValue = state.search || '';
    if (ui.searchInput.value !== currentValue) {
      ui.searchInput.value = currentValue;
    }
    ui.searchInput.disabled = !isCustom;
    if (isCustom) {
      ui.searchInput.removeAttribute('title');
    } else {
      ui.searchInput.title = '선택한 항목만 모드에서 사용할 수 있습니다';
    }
  }
  if (ui.refreshButton) {
    ui.refreshButton.disabled = !isCustom;
    if (isCustom) {
      ui.refreshButton.removeAttribute('title');
    } else {
      ui.refreshButton.title = '선택한 항목만 모드에서만 새로고침할 수 있습니다';
    }
  }
}

function renderResourceModeControls(adminId, type) {
  const ui = getResourceUIElements();
  if (!ui?.modeContainer) return;
  const state = ensureAdminResourceState(adminId)[type];
  const modes = [
    {
      value: 'all',
      label: '모두 허용',
      description: '기능 권한이 허용한 범위 내에서 모든 항목을 열람·관리할 수 있습니다.',
    },
    {
      value: 'none',
      label: '모두 차단',
      description: '이 리소스는 관리자에게 표시되지 않습니다.',
    },
    {
      value: 'own',
      label: '내가 생성한 항목',
      description: '이 관리자가 직접 생성한 항목만 열람·관리할 수 있습니다.',
    },
    {
      value: 'custom',
      label: '선택한 항목만',
      description: '검색과 체크박스를 사용해 허용할 항목을 지정하세요.',
    },
  ];
  ui.modeContainer.innerHTML = modes
    .map(
      (mode) => `
    <label class="resource-mode-card ${state.mode === mode.value ? 'is-active' : ''}">
      <input type="radio" name="resource-mode-${type}" value="${mode.value}" ${state.mode === mode.value ? 'checked' : ''}>
      <span class="resource-mode-title">${mode.label}</span>
      <span class="resource-mode-description">${mode.description}</span>
    </label>
  `,
    )
    .join('');
}

function getFilteredResourceItems(items, searchTerm) {
  if (!searchTerm) return items;
  const lc = searchTerm.toLowerCase();
  return items.filter((item) => {
    return (
      (item.name && item.name.toLowerCase().includes(lc)) ||
      (item.description && item.description.toLowerCase().includes(lc))
    );
  });
}

async function renderResourceList(adminId, type) {
  const ui = getResourceUIElements();
  if (!ui?.list) return;
  const state = ensureAdminResourceState(adminId)[type];
  ui.list.innerHTML = '';

  if (state.mode !== 'custom') {
    ui.list.innerHTML =
      '<div class="resource-empty">"선택한 항목만" 모드에서 허용할 항목을 선택할 수 있습니다.</div>';
    return;
  }

  const catalog = await loadResourceCatalog(type);
  if (catalog.error) {
    ui.list.innerHTML = `<div class="resource-empty">${escapeHtml(catalog.error)}</div>`;
    return;
  }

  const items = getFilteredResourceItems(catalog.items || [], state.search);
  if (!items.length) {
    ui.list.innerHTML = '<div class="resource-empty">조건에 맞는 항목이 없습니다.</div>';
    return;
  }

  const rows = items
    .map((item) => {
      const checkboxId = `resource-${type}-${item.id}`;
      const isSelected = state.selected.has(item.id);
      return `
        <tr class="resource-table-row ${isSelected ? 'is-selected' : ''}" data-resource-id="${item.id}">
          <td class="resource-table-check">
            <input type="checkbox" class="resource-item-checkbox" id="${checkboxId}" ${isSelected ? 'checked' : ''} />
          </td>
          <td class="resource-table-info">
            <label for="${checkboxId}">
              <div class="resource-item-title">${escapeHtml(item.name || '-')}</div>
              <div class="resource-item-meta">${escapeHtml(item.description || '')}</div>
            </label>
          </td>
        </tr>
      `;
    })
    .join('');

  ui.list.innerHTML = `
    <table class="resource-table">
      <colgroup>
        <col class="resource-table-col-check" />
        <col />
      </colgroup>
      <thead>
        <tr>
          <th scope="col" class="resource-table-header-check">선택</th>
          <th scope="col">항목</th>
        </tr>
      </thead>
      <tbody>
        ${rows}
      </tbody>
    </table>
  `;
}

function renderResourceHelper(adminId, type) {
  const ui = getResourceUIElements();
  if (!ui?.helper) return;
  const state = ensureAdminResourceState(adminId)[type];
  const config = resourceTypeIndex.get(type);
  let message = '';
  let variant = 'info';

  switch (state.mode) {
    case 'none':
      message = `${config?.label || type} 리소스가 관리자에게 표시되지 않습니다.`;
      variant = 'warn';
      break;
    case 'own':
      message = '이 관리자가 직접 생성한 항목만 열람·관리할 수 있습니다.';
      variant = 'success';
      break;
    case 'custom':
      message = '허용할 항목을 체크박스로 선택하세요. 선택하지 않으면 접근이 제한됩니다.';
      variant = 'info';
      break;
    case 'all':
    default:
      message = `${config?.label || type}의 모든 항목에 접근할 수 있습니다.`;
      variant = 'info';
      break;
  }

  ui.helper.textContent = message;
  ui.helper.dataset.variant = variant;
}

function renderResourceSummary(adminId) {
  const ui = getResourceUIElements();
  if (!ui?.summary) return;
  const state = ensureAdminResourceState(adminId);

  const items = RESOURCE_TYPES.map((type) => {
    const entry = state[type.key] || { mode: 'all', selected: new Set() };
    const config = resourceTypeIndex.get(type.key);
    const chipClass = getResourceModeChipClass(entry.mode);
    const modeLabel = getAdminResourceModeLabel(entry.mode);
    const selectionCount = entry.selected instanceof Set ? entry.selected.size : 0;
    let detail = '';
    switch (entry.mode) {
      case 'custom':
        detail = selectionCount > 0 ? `선택 ${selectionCount}개` : '선택된 항목 없음';
        break;
      case 'own':
        detail = '현재 관리자 생성분';
        break;
      case 'none':
        detail = '접근 차단';
        break;
      default:
        detail = '';
    }

    return `
      <div class="resource-summary-item">
        <span class="resource-summary-item-label">${escapeHtml(config?.summaryLabel || config?.label || type.label || type.key)}</span>
        <div class="resource-summary-item-detail">
          <span class="${chipClass}">${escapeHtml(modeLabel)}</span>
          ${detail ? `<span class="resource-summary-note">${escapeHtml(detail)}</span>` : ''}
        </div>
      </div>
    `;
  }).join('');

  ui.summary.innerHTML = `
    <div>
      <h4 class="resource-summary-title">현재 설정</h4>
      <p class="resource-summary-note">리소스 권한은 기능 권한 범위 안에서만 적용됩니다.</p>
    </div>
    <div class="resource-summary-list">
      ${items}
    </div>
  `;
}

async function renderResourcePermissionsPane(adminId) {
  const ui = getResourceUIElements();
  if (!ui) return;
  resourceUIState.activeAdminId = adminId;
  if (!resourceUIState.activeType) {
    resourceUIState.activeType = RESOURCE_TYPES[0]?.key || null;
  }

  renderResourceTypeTabs(adminId);
  const activeType = resourceUIState.activeType;
  updateResourceToolsState(adminId, activeType);
  renderResourceModeControls(adminId, activeType);
  renderResourceHelper(adminId, activeType);
  await renderResourceList(adminId, activeType);
  renderResourceSummary(adminId);
}

function handleResourceTypeTabClick(event) {
  const button = event.target.closest('button[data-resource-type]');
  if (!button) return;
  const { resourceType } = button.dataset;
  if (!resourceTypeIndex.has(resourceType)) return;
  resourceUIState.activeType = resourceType;
  const adminId = resourceUIState.activeAdminId;
  if (!adminId) return;
  renderResourcePermissionsPane(adminId);
}

function handleResourceModeChange(event) {
  if (event.target.tagName !== 'INPUT') return;
  const { value } = event.target;
  const type = resourceUIState.activeType;
  const adminId = resourceUIState.activeAdminId;
  if (!type || !adminId) return;
  const state = ensureAdminResourceState(adminId)[type];
  state.mode = value;
  if (value !== 'custom') {
    state.selected.clear();
  }
  renderResourceModeControls(adminId, type);
  updateResourceToolsState(adminId, type);
  renderResourceHelper(adminId, type);
  renderResourceSummary(adminId);
  renderResourceList(adminId, type);
  renderResourceTypeTabs(adminId);
}

function handleResourceItemToggle(event) {
  const row = event.target.closest('.resource-table-row');
  if (!row) return;
  const checkbox = row.querySelector('.resource-item-checkbox');
  if (!checkbox) return;
  if (event.target === checkbox || event.target.tagName === 'LABEL') {
    return;
  }
  checkbox.click();
}

function handleResourceItemCheckboxChange(event) {
  const checkbox = event.target.closest('.resource-item-checkbox');
  if (!checkbox) return;
  const row = checkbox.closest('.resource-table-row');
  if (!row) return;
  const type = resourceUIState.activeType;
  const adminId = resourceUIState.activeAdminId;
  if (!type || !adminId) return;
  const state = ensureAdminResourceState(adminId)[type];
  if (state.mode !== 'custom') {
    checkbox.checked = false;
    return;
  }
  const resourceId = row.dataset.resourceId;
  if (!resourceId) return;
  if (checkbox.checked) {
    state.selected.add(resourceId);
  } else {
    state.selected.delete(resourceId);
  }
  row.classList.toggle('is-selected', checkbox.checked);
  renderResourceSummary(adminId);
  renderResourceTypeTabs(adminId);
}

function handleResourceSearchInput(event) {
  const type = resourceUIState.activeType;
  const adminId = resourceUIState.activeAdminId;
  if (!type || !adminId) return;
  const state = ensureAdminResourceState(adminId)[type];
  if (state.mode !== 'custom') return;
  state.search = event.target.value.trim();
  renderResourceList(adminId, type);
}

function handleResourceRefresh() {
  const type = resourceUIState.activeType;
  const adminId = resourceUIState.activeAdminId;
  if (!type || !adminId) return;
  const state = ensureAdminResourceState(adminId)[type];
  if (state.mode !== 'custom') return;
  resourceCatalogCache.delete(type);
  renderResourceList(adminId, type);
}

function initializeResourcePermissionEvents() {
  const ui = getResourceUIElements();
  if (!ui) return;
  if (ui.typeTabs && !ui.typeTabs.dataset.bound) {
    ui.typeTabs.addEventListener('click', handleResourceTypeTabClick);
    ui.typeTabs.dataset.bound = 'true';
  }
  if (ui.modeContainer && !ui.modeContainer.dataset.bound) {
    ui.modeContainer.addEventListener('change', handleResourceModeChange);
    ui.modeContainer.dataset.bound = 'true';
  }
  if (ui.list && !ui.list.dataset.bound) {
    ui.list.addEventListener('click', handleResourceItemToggle);
    ui.list.addEventListener('change', handleResourceItemCheckboxChange);
    ui.list.dataset.bound = 'true';
  }
  if (ui.searchInput && !ui.searchInput.dataset.bound) {
    ui.searchInput.addEventListener('input', handleResourceSearchInput);
    ui.searchInput.dataset.bound = 'true';
  }
  if (ui.refreshButton && !ui.refreshButton.dataset.bound) {
    ui.refreshButton.addEventListener('click', handleResourceRefresh);
    ui.refreshButton.dataset.bound = 'true';
  }
}

function switchPermissionPane(target) {
  const modal = document.getElementById('manage-admin-permissions-modal');
  if (!modal) return;
  const functionPane = modal.querySelector('#function-permission-pane');
  const resourcePane = modal.querySelector('#resource-permission-pane');
  const showResource = target === 'resource';

  if (functionPane) {
    const isActive = !showResource;
    functionPane.classList.toggle('is-active', isActive);
    functionPane.hidden = !isActive;
  }
  if (resourcePane) {
    resourcePane.classList.toggle('is-active', showResource);
    resourcePane.hidden = !showResource;
  }

  modal.setAttribute('data-active-permission-pane', showResource ? 'resource' : 'function');

  if (showResource && resourceUIState.activeAdminId) {
    renderResourcePermissionsPane(resourceUIState.activeAdminId);
  }
}

function updatePermissionGroupSummary(group) {
  const summaryEl = group.querySelector('.permission-group-summary');
  if (!summaryEl) return;

  const checkboxes = Array.from(group.querySelectorAll('input[data-permission-key]'));
  const selectedCount = checkboxes.filter((input) => input.checked).length;
  summaryEl.textContent = `선택 ${selectedCount} / ${checkboxes.length}`;

  const icon = group.querySelector('.permission-group-icon');
  if (icon) {
    icon.textContent = group.classList.contains('collapsed') ? '▶' : '▼';
  }

  const body = group.querySelector('.permission-group-body');
  if (body) {
    body.style.display = group.classList.contains('collapsed') ? 'none' : 'flex';
  }
}

function getSelectedPermissions(container) {
  if (!container) return [];
  return Array.from(container.querySelectorAll('input[data-permission-key]:checked')).map((input) => input.value);
}

function getPermissionLabel(key) {
  const found = permissionCatalog.find((item) => item.key === key);
  return found?.label || key;
}

function buildPermissionSummary(permissionKeys, isSuper) {
  if (isSuper) {
    return '<span class="permission-badge permission-badge--all"><span class="permission-badge-icon">✔</span>모든 권한</span>';
  }

  if (!permissionKeys || permissionKeys.length === 0) {
    return '<span class="permission-badge permission-badge--empty"><span class="permission-badge-icon">–</span>없음</span>';
  }

  const labels = permissionKeys.map((key) => escapeHtml(getPermissionLabel(key)));
  const fragments = [];
  const visibleCount = 2;

  labels.slice(0, visibleCount).forEach((label) => {
    fragments.push(`<span class="permission-badge"><span class="permission-badge-icon">✔</span>${label}</span>`);
  });

  if (labels.length > visibleCount) {
    fragments.push(`<span class="permission-badge permission-badge--more">+${labels.length - visibleCount}</span>`);
  }

  return fragments.join('');
}

export async function loadAdmins() {
  try {
    const tbody = document.getElementById('admins-tbody');
    if (!tbody) {
      console.error('admins-tbody element not found');
      return;
    }
    // 로딩 상태 표시 (요청 시작 전)
    tbody.innerHTML = '<tr><td colspan="6" class="text-center">로딩 중...</td></tr>';

    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const body = await res.json();
    
    if (res.ok && body.status === 'success') {
  const admins = body.data || [];
  console.log('Loaded admins:', admins);
  try {
    await ensurePermissionCatalog();
  } catch (err) {
    console.warn('Permission catalog unavailable:', err);
  }
  adminsCache.clear();

  // 역할 정규화 헬퍼
  const isSuper = (role) => {
    if (!role) return false;
    return String(role).toLowerCase().replace(/-/g, '_') === 'super_admin';
      };
      
      if (admins.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center">관리자가 없습니다.</td></tr>';
      } else {
        // DOM API로 안전하게 렌더링하여 셀 누락 문제를 방지
        tbody.innerHTML = '';
        admins.forEach((a) => {
          applyResourcePermissionsToState(a.id, a.resource_permissions);
          const cachedAdmin = {
            ...a,
            resource_permissions: serializeResourcePermissions(a.id),
          };
          adminsCache.set(String(a.id), cachedAdmin);
          const tr = document.createElement('tr');

          // 아이디/유저명
          const tdUser = document.createElement('td');
          tdUser.innerHTML = `${escapeHtml(a.username)} <small class="mono" style="color:#777;">(${escapeHtml(a.id)})</small>`;
          tr.appendChild(tdUser);

          // 이메일
          const tdEmail = document.createElement('td');
          tdEmail.textContent = a.email ? String(a.email) : '-';
          tr.appendChild(tdEmail);

          // 역할 배지
          const tdRole = document.createElement('td');
          const roleSpan = document.createElement('span');
          roleSpan.className = `role-badge ${isSuper(a.role) ? 'super' : 'admin'}`;
          const iconSpan = document.createElement('span');
          iconSpan.className = 'icon';
          iconSpan.textContent = isSuper(a.role) ? '⭐' : '👤';
          roleSpan.appendChild(iconSpan);
          roleSpan.appendChild(document.createTextNode(` ${isSuper(a.role) ? 'Super Admin' : 'Admin'}`));
          tdRole.appendChild(roleSpan);
          tr.appendChild(tdRole);

          const permissionKeys = Array.isArray(cachedAdmin.permissions) ? cachedAdmin.permissions : [];
          const tdPermissions = document.createElement('td');
          tdPermissions.className = 'admin-permissions-cell';
          tdPermissions.innerHTML = buildPermissionSummary(permissionKeys, isSuper(a.role));
          tr.appendChild(tdPermissions);

          // 생성일
          const tdCreated = document.createElement('td');
          tdCreated.textContent = formatDateTime(a.created_at);
          tr.appendChild(tdCreated);

          // 작업
          const tdActions = document.createElement('td');
          const actionsDiv = document.createElement('div');
          actionsDiv.className = 'actions-cell';
          if (isSuper(a.role)) {
            const disabledA = document.createElement('a');
            disabledA.href = '#';
            disabledA.className = 'btn btn-sm btn-warning disabled';
            disabledA.setAttribute('aria-disabled', 'true');
            disabledA.title = '슈퍼 관리자는 비활성화됨';
            disabledA.textContent = '🔒 초기화 불가';
            actionsDiv.appendChild(disabledA);
          } else {
            const manageBtn = document.createElement('a');
            manageBtn.href = '#';
            manageBtn.className = 'btn btn-sm grey lighten-1';
            manageBtn.dataset.action = 'permissions';
            manageBtn.dataset.adminId = String(a.id);
            manageBtn.dataset.adminName = String(a.username);
            manageBtn.dataset.permissions = permissionKeys.join(',');
            manageBtn.textContent = '⚙️ 기능 권한';
            actionsDiv.appendChild(manageBtn);

            const resourceBtn = document.createElement('a');
            resourceBtn.href = '#';
            resourceBtn.className = 'btn btn-sm indigo lighten-1';
            resourceBtn.dataset.action = 'resource-permissions';
            resourceBtn.dataset.adminId = String(a.id);
            resourceBtn.dataset.adminName = String(a.username);
            resourceBtn.dataset.permissions = permissionKeys.join(',');
            resourceBtn.textContent = '🗂️ 리소스 권한';
            actionsDiv.appendChild(resourceBtn);

            const resetA = document.createElement('a');
            resetA.href = '#';
            resetA.className = 'btn btn-sm btn-warning';
            resetA.dataset.action = 'reset';
            resetA.dataset.adminId = String(a.id);
            resetA.dataset.adminName = String(a.username);
            resetA.textContent = '🔑 비밀번호 초기화';

            const delA = document.createElement('a');
            delA.href = '#';
            delA.className = 'btn btn-sm btn-danger';
            delA.dataset.action = 'delete';
            delA.dataset.adminId = String(a.id);
            delA.dataset.adminName = String(a.username);
            delA.textContent = '🗑️ 삭제';

            actionsDiv.appendChild(resetA);
            actionsDiv.appendChild(delA);
          }
          tdActions.appendChild(actionsDiv);
          tr.appendChild(tdActions);

          tbody.appendChild(tr);
        });

        console.log('Admin table updated successfully (DOM render)');
      }
    } else {
      tbody.innerHTML = `<tr><td colspan="6" class="text-center">불러오기에 실패했습니다: ${escapeHtml(body.message || '')}</td></tr>`;
    }
  } catch (e) {
    console.error('Failed to load admins:', e);
    const tbody = document.getElementById('admins-tbody');
    if (tbody) tbody.innerHTML = '<tr><td colspan="6" class="text-center">서버 오류</td></tr>';
  }
}

export async function handleCreateAdmin(e) {
  e.preventDefault();
  const username = document.getElementById('admin_username').value.trim();
  const email = document.getElementById('admin_email').value.trim();
  const password = document.getElementById('admin_password').value;
  if (!username || !email || !password) return;

  const submitBtn = e.target.querySelector('button[type="submit"]');
  const originalBtnText = submitBtn ? submitBtn.textContent : '';
  const originalBtnDisabled = submitBtn ? submitBtn.disabled : false;

  if (submitBtn) {
    submitBtn.disabled = true;
    submitBtn.textContent = '생성 중...';
  }

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins/create`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${state.token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
      _noGlobalLoading: true
    });
    const body = await res.json();

    if (res.ok && body.status === 'success') {
      await loadAdmins();
      if (window.loadRecentActivities) await window.loadRecentActivities();

      const createAdminModal = document.getElementById('create-admin-modal');
      if (createAdminModal) {
        closeModal(createAdminModal);
      }
      e.target.reset();

      if (submitBtn) {
        submitBtn.disabled = originalBtnDisabled;
        submitBtn.textContent = originalBtnText;
      }

      setTimeout(() => {
        showAlert('서브 관리자가 생성되었습니다.\n\n생성된 계정은 권한이 없습니다. 관리자 목록에서 기능 권한과 리소스 권한을 수동으로 부여하세요.', '관리자 생성');
      }, 300);
    } else {
      const createAdminModal = document.getElementById('create-admin-modal');
      if (createAdminModal) {
        closeModal(createAdminModal);
      }
      e.target.reset();

      if (submitBtn) {
        submitBtn.disabled = originalBtnDisabled;
        submitBtn.textContent = originalBtnText;
      }

      setTimeout(() => {
        showAlert(body.message || '생성에 실패했습니다.', '관리자 생성 실패');
      }, 300);
      return;
    }
  } catch (err) {
    console.error('Failed to create admin:', err);

    const createAdminModal = document.getElementById('create-admin-modal');
    if (createAdminModal) {
      closeModal(createAdminModal);
    }
    e.target.reset();

    if (submitBtn) {
      submitBtn.disabled = originalBtnDisabled;
      submitBtn.textContent = originalBtnText;
    }

    setTimeout(() => {
      showAlert('서버 오류가 발생했습니다.', '관리자 생성 실패');
    }, 300);
    return;
  }
}

export async function prepareCreateAdminModal() {
  const form = document.getElementById('create-admin-form');
  if (form) {
    form.reset();
  }
  const usernameInput = document.getElementById('admin_username');
  if (usernameInput) {
    usernameInput.focus();
  }
}

async function openManagePermissionsModal(adminId, adminName, permissions = [], options = {}) {
  const initialTab = options?.initialTab === 'resource' ? 'resource' : 'function';
  try {
    await ensurePermissionCatalog();
  } catch (err) {
    console.warn('Permission catalog unavailable for manage modal:', err);
  }

  const modal = document.getElementById('manage-admin-permissions-modal');
  const container = document.getElementById('manage-admin-permissions');
  const hiddenId = document.getElementById('manage-admin-id');
  const nameEl = document.getElementById('manage-admin-name');

  if (hiddenId) hiddenId.value = adminId;
  if (nameEl) nameEl.textContent = adminName || '-';

  const cached = adminsCache.get(String(adminId));
  let effectivePermissions = [];
  if (cached?.permissions && Array.isArray(cached.permissions) && cached.permissions.length > 0) {
    effectivePermissions = [...cached.permissions];
  } else if (Array.isArray(permissions) && permissions.length > 0) {
    effectivePermissions = [...permissions];
  }
  applyResourcePermissionsToState(adminId, cached?.resource_permissions);

  renderPermissionChecklist(container, effectivePermissions);

  ensureAdminResourceState(adminId);
  const previousAdmin = resourceUIState.activeAdminId;
  resourceUIState.activeAdminId = adminId;
  if (previousAdmin !== adminId && RESOURCE_TYPES.length) {
    resourceUIState.activeType = RESOURCE_TYPES[0].key;
  }
  initializeResourcePermissionEvents();
  switchPermissionPane(initialTab);
  renderResourceSummary(adminId);

  if (modal) {
    openModal(modal);
  }
}

async function handleUpdateAdminPermissions(e) {
  e.preventDefault();
  const adminId = document.getElementById('manage-admin-id')?.value;
  if (!adminId) return;

  const container = document.getElementById('manage-admin-permissions');
  const selectedPermissions = getSelectedPermissions(container);
  const submitBtn = e.target.querySelector('button[type="submit"]');
  const originalText = submitBtn ? submitBtn.textContent : '';
  const originalDisabled = submitBtn ? submitBtn.disabled : false;

  if (submitBtn) {
    submitBtn.disabled = true;
    submitBtn.textContent = '저장 중...';
  }

  try {
    const resourcePayload = serializeResourcePermissions(adminId);
    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins/${adminId}/permissions`, {
      method: 'PUT',
      headers: { 'Authorization': `Bearer ${state.token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        permissions: selectedPermissions,
        resource_permissions: resourcePayload,
      }),
      _noGlobalLoading: true
    });
    const body = await res.json();

    if (res.ok && body.status === 'success') {
      const modal = document.getElementById('manage-admin-permissions-modal');
      if (modal) closeModal(modal);

      const cached = adminsCache.get(String(adminId)) || {};
      const responseResource = body.data?.resource_permissions || resourcePayload;
      applyResourcePermissionsToState(adminId, responseResource);
      adminsCache.set(String(adminId), {
        ...cached,
        permissions: [...selectedPermissions],
        resource_permissions: serializeResourcePermissions(adminId),
      });

      await loadAdmins();
      if (window.loadRecentActivities) window.loadRecentActivities();

      setTimeout(() => {
        showAlert('관리자 권한이 업데이트되었습니다.', '권한 업데이트');
      }, 200);
    } else {
      showAlert(body.message || '권한을 업데이트하지 못했습니다.', '권한 업데이트 실패');
    }
  } catch (err) {
    console.error('Failed to update admin permissions:', err);
    showAlert('권한을 업데이트하는 중 서버 오류가 발생했습니다.', '권한 업데이트 실패');
  } finally {
    if (submitBtn) {
      submitBtn.disabled = originalDisabled;
      submitBtn.textContent = originalText || '저장';
    }
  }
}

// 비밀번호 초기화
async function resetAdminPassword(adminId, adminUsername, btn) {
  const ok = await showConfirm(`${adminUsername}의 비밀번호를 초기화하시겠습니까?\n\n임시 비밀번호가 생성됩니다. 본인이 직접 변경하도록 안내하세요.`, '비밀번호 초기화 확인');
  if (!ok) return;

  const originalText = btn?.textContent || '';
  if (btn) {
    btn.disabled = true;
    btn.textContent = '초기화 중...';
  }

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins/${adminId}/reset-password`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${state.token}`, 'Content-Type': 'application/json' },
      _noGlobalLoading: true
    });
    const body = await res.json();
    if (res.ok && body.status === 'success') {
      const tempPassword = body.data?.temp_password || 'N/A';
      // 버튼 복구
      if (btn) {
        btn.disabled = false;
        btn.textContent = originalText;
      }
      await showAlert(`비밀번호가 초기화되었습니다.\n\n임시 비밀번호: ${tempPassword}\n\n이 임시 비밀번호를 ${adminUsername}에게 전달하세요.`, '비밀번호 초기화 완료');
      loadAdmins();
      if (window.loadRecentActivities) window.loadRecentActivities();
    } else {
      // 버튼 복구
      if (btn) {
        btn.disabled = false;
        btn.textContent = originalText;
      }
      await showAlert(body.message || '초기화에 실패했습니다.', '비밀번호 초기화 실패');
    }
  } catch (err) {
    console.error('Failed to reset admin password:', err);
    // 버튼 복구
    if (btn) {
      btn.disabled = false;
      btn.textContent = originalText;
    }
    await showAlert('서버 오류가 발생했습니다.', '비밀번호 초기화 실패');
  }
}

// 관리자 계정 삭제
async function deleteAdmin(adminId, adminUsername, btn) {
  const ok = await showConfirm(`정말 ${adminUsername} 계정을 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.`, '관리자 삭제 확인');
  if (!ok) return;

  const originalText = btn?.textContent || '';
  if (btn) {
    btn.disabled = true;
    btn.textContent = '삭제 중...';
  }

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins/${adminId}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${state.token}` },
      _noGlobalLoading: true
    });
    const body = await res.json();
    if (res.ok && body.status === 'success') {
      // 버튼 복구
      if (btn) {
        btn.disabled = false;
        btn.textContent = originalText;
      }
      await showAlert('관리자가 삭제되었습니다.', '관리자 삭제 완료');
      loadAdmins();
      if (window.loadRecentActivities) window.loadRecentActivities();
    } else {
      // 버튼 복구
      if (btn) {
        btn.disabled = false;
        btn.textContent = originalText;
      }
      await showAlert(body.message || '삭제에 실패했습니다.', '관리자 삭제 실패');
    }
  } catch (err) {
    console.error('Failed to delete admin:', err);
    // 버튼 복구
    if (btn) {
      btn.disabled = false;
      btn.textContent = originalText;
    }
    await showAlert('서버 오류가 발생했습니다.', '관리자 삭제 실패');
  }
}

// 전역 스코프에 노출
window.prepareCreateAdminModal = prepareCreateAdminModal;
window.openManagePermissionsModal = openManagePermissionsModal;
window.resetAdminPassword = resetAdminPassword;
window.deleteAdmin = deleteAdmin;

const managePermissionsForm = document.getElementById('manage-admin-permissions-form');
if (managePermissionsForm) {
  managePermissionsForm.addEventListener('submit', handleUpdateAdminPermissions);
}
