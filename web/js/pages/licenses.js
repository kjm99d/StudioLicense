import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { formatDate } from '../utils.js';
import { renderStatusBadge } from '../ui.js';

let productsCached = null; // 제품 목록 캐시

export async function loadLicenses(page = 1) {
  try {
    let url = `${API_BASE_URL}/api/admin/licenses?page=${page}&page_size=10`;
    if (state.currentStatus) url += `&status=${state.currentStatus}`;
    if (state.currentSearch) url += `&search=${encodeURIComponent(state.currentSearch)}`;

    const response = await apiFetch(url, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const data = await response.json();

    console.log('API Response licenses count:', data.data?.length || 0);
    
    if (data.status === 'success') {
      renderLicensesTable(data.data);
      renderPagination(data.meta);
    }
  } catch (error) {
    console.error('Failed to load licenses:', error);
  }
}

function renderLicensesTable(licenses) {
  const tbody = document.getElementById('licenses-tbody');
  if (!licenses || licenses.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" class="loading">라이선스가 없습니다.</td></tr>';
    return;
  }
  console.log('Rendering licenses table with data:', licenses);
  tbody.innerHTML = licenses.map(license => {
    const statusHtml = renderStatusBadge(license.status);
    console.log(`License ${license.id} status: ${license.status} -> HTML: ${statusHtml.substring(0, 80)}`);
    return `
    <tr>
      <td><code>${license.license_key}</code></td>
      <td>${license.product_name}</td>
      <td>${license.customer_name}</td>
      <td>${license.max_devices}</td>
      <td>${formatDate(license.expires_at)}</td>
      <td>${statusHtml}</td>
      <td>
        <button class="btn btn-sm" onclick="viewLicense('${license.id}')">상세</button>
        <button class="btn btn-sm btn-danger" onclick="deleteLicense('${license.id}')">삭제</button>
      </td>
    </tr>
  `;
  }).join('');
}

function renderPagination(meta) {
  const container = document.getElementById('pagination');
  if (!meta || meta.total_pages <= 1) {
    container.innerHTML = '';
    return;
  }
  let html = '';
  for (let i = 1; i <= meta.total_pages; i++) {
    html += `<button class="${i === meta.page ? 'active' : ''}" onclick="loadLicenses(${i})">${i}</button>`;
  }
  container.innerHTML = html;
}

export function openLicenseModal() {
  const nextYear = new Date();
  nextYear.setFullYear(nextYear.getFullYear() + 1);
  document.getElementById('expires_at').value = nextYear.toISOString().split('T')[0];
  const today = new Date();
  document.getElementById('expires_at').setAttribute('min', today.toISOString().split('T')[0]);
  populateProductDropdown();
  openModal(document.getElementById('license-modal'));
}

export async function handleCreateLicense(e) {
  console.log('handleCreateLicense called', e);
  e.preventDefault();
  const formData = new FormData(e.target);
  let data = {
    product_id: formData.get('product_id') || '',
    customer_name: formData.get('customer_name'),
    customer_email: formData.get('customer_email'),
    max_devices: parseInt(formData.get('max_devices')),
    expires_at: new Date(formData.get('expires_at')).toISOString(),
    notes: formData.get('notes')
  };

  if (!data.product_id) {
    const sel = document.getElementById('product_select');
    if (sel && sel.value) data.product_id = sel.value;
  }

  const dateInput = document.getElementById('expires_at').value;
  const selectedDate = new Date(dateInput + 'T00:00:00');
  const startOfToday = new Date(); startOfToday.setHours(0,0,0,0);
  if (selectedDate < startOfToday) {
    await showAlert('만료일은 과거로 설정할 수 없습니다.', '라이선스 생성');
    return;
  }

  const submitBtn = e.target.querySelector('button[type="submit"]');
  const originalBtnText = submitBtn ? submitBtn.textContent : '';
  if (submitBtn) { submitBtn.disabled = true; submitBtn.textContent = '생성 중...'; }

  try {
    console.log('Sending license creation request:', data);
    const response = await apiFetch(`${API_BASE_URL}/api/admin/licenses`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${state.token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
      _noGlobalLoading: true
    });
    const result = await response.json();
    console.log('License creation response:', result);
    if (result.status === 'success') {
      const licenseModal = document.getElementById('license-modal');
      if (licenseModal) closeModal(licenseModal);
      e.target.reset();
      
      // alert 후에 데이터 로드
      await showAlert(`라이선스가 생성되었습니다!\n라이선스 키: ${result.data.license_key}`, '라이선스 생성');
      
      // alert이 닫힌 후 데이터 새로고침
      setTimeout(() => {
        loadLicenses();
        if (window.loadDashboardStats) window.loadDashboardStats();
      }, 100);
    } else {
      await showAlert('라이선스 생성 실패: ' + result.message, '라이선스 생성');
    }
  } catch (error) {
    await showAlert('서버 오류가 발생했습니다.', '라이선스 생성');
    console.error('Failed to create license:', error);
  } finally {
    if (submitBtn) { submitBtn.disabled = false; submitBtn.textContent = originalBtnText; }
  }
}

async function populateProductDropdown() {
  const select = document.getElementById('product_select');
  if (!select) return;
  
  try {
    // 캐시가 없으면 API 호출
    if (!productsCached) {
      const res = await apiFetch(`${API_BASE_URL}/api/admin/products?status=active`, { headers: { 'Authorization': `Bearer ${state.token}` } });
      if (!res.ok) throw new Error('Failed to load products');
      const body = await res.json();
      productsCached = body.data || [];
      console.log('Products loaded from API:', productsCached.length);
    } else {
      console.log('Using cached products:', productsCached.length);
    }

    select.innerHTML = '';
    productsCached.forEach(p => {
      const opt = document.createElement('option');
      opt.value = p.id;
      opt.textContent = `${p.name}`;
      select.appendChild(opt);
    });
    
    let hiddenId = document.getElementById('product_id_hidden');
    select.onchange = () => {
      if (!hiddenId) return;
      const selected = select.selectedOptions[0];
      hiddenId.value = selected && selected.value ? selected.value : '';
    };
    if (productsCached.length > 0) select.value = productsCached[0].id;
    select.required = true;
    select.onchange();
  } catch (err) {
    console.error('Failed to populate product dropdown:', err);
  }
}

export async function viewLicense(id) {
  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/licenses/?id=${id}`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const data = await response.json();
    if (data.status === 'success') {
      const license = data.data;
      const content = `
        <div class="detail-group"><div class="detail-label">라이선스 키</div><div class="detail-value"><code>${license.license_key}</code></div></div>
        <div class="detail-group"><div class="detail-label">제품명</div><div class="detail-value">${license.product_name}</div></div>
        <div class="detail-group"><div class="detail-label">고객 정보</div><div class="detail-value">${license.customer_name}<br>${license.customer_email}</div></div>
        <div class="detail-group"><div class="detail-label">최대 디바이스</div><div class="detail-value">${license.max_devices}</div></div>
        <div class="detail-group"><div class="detail-label">만료일</div><div class="detail-value">${formatDate(license.expires_at)}</div></div>
        <div class="detail-group"><div class="detail-label">상태</div><div class="detail-value">${renderStatusBadge(license.status)}</div></div>
        <div class="detail-group"><div class="detail-label">메모</div><div class="detail-value">${license.notes || '-'}</div></div>
        <div class="detail-group"><div class="detail-label">등록된 디바이스</div><div class="detail-value"><div id="license-devices" class="device-list loading">불러오는 중...</div></div></div>
      `;
      document.getElementById('license-detail-content').innerHTML = content;
      openModal(document.getElementById('license-detail-modal'));
      loadDevicesForLicense(id);
    }
  } catch (error) {
    console.error('Failed to load license:', error);
  }
}

async function loadDevicesForLicense(id) {
  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/licenses/devices?id=${id}`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const body = await res.json();
    const container = document.getElementById('license-devices');
    if (res.ok && body.status === 'success') {
      const devices = body.data || [];
      container.classList.remove('loading');
      if (devices.length === 0) {
        container.textContent = '등록된 디바이스가 없습니다.';
      } else {
        // renderDeviceCard is in app.js still, keep global call
        container.innerHTML = devices.map(d => window.renderDeviceCard ? window.renderDeviceCard(d) : '').join('');
      }
    } else {
      container.classList.remove('loading');
      container.textContent = body.message || '디바이스 정보를 불러오지 못했습니다.';
    }
  } catch (e) {
    const container = document.getElementById('license-devices');
    if (container) { container.classList.remove('loading'); container.textContent = '디바이스 정보를 불러오지 못했습니다.'; }
  }
}

export async function deleteLicense(id) {
  const ok = await showConfirm('정말로 이 라이선스를 삭제하시겠습니까?', '라이선스 삭제');
  if (!ok) return;
  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/licenses/?id=${id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${state.token}` }
    });
    const data = await response.json();
    if (data.status === 'success') {
      await showAlert('라이선스가 삭제되었습니다.', '라이선스 삭제');
      loadLicenses();
      if (window.loadDashboardStats) window.loadDashboardStats();
    } else {
      await showAlert('삭제 실패: ' + data.message, '라이선스 삭제');
    }
  } catch (error) {
    await showAlert('서버 오류가 발생했습니다.', '라이선스 삭제');
    console.error('Failed to delete license:', error);
  }
}

export function handleSearch(e) {
  state.currentSearch = e.target.value;
  state.currentPage = 1;
  loadLicenses();
}

export function handleFilter(e) {
  state.currentStatus = e.target.value;
  state.currentPage = 1;
  loadLicenses();
}
