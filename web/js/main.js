import { state } from './state.js';
import { apiFetch, API_BASE_URL } from './api.js';
import { setupModalBehaviors, openModal, closeModal, showAlert, showConfirm } from './modals.js';
import { formatDate, formatDateTime, debounce } from './utils.js';
import { renderStatusBadge } from './ui.js';
import { loadLicenses, openLicenseModal, handleCreateLicense, viewLicense, deleteLicense, handleSearch, handleFilter } from './pages/licenses.js';
import { loadAdmins, handleCreateAdmin } from './pages/admins.js';
import { loadDashboardStats, loadRecentActivities } from './pages/dashboard.js';
import './pages/products.js'; // 제품 관리 페이지

// Expose helpers globally for legacy scripts (products.js, device rendering still in app.js)
window.apiFetch = apiFetch;
window.API_BASE_URL = API_BASE_URL;
window.formatDate = formatDate;
window.formatDateTime = formatDateTime;
window.showAlert = showAlert;
window.showConfirm = showConfirm;
window.openModal = openModal;
window.closeModal = closeModal;
window.renderStatusBadge = renderStatusBadge;
window.state = state; // state 전역 노출
window.token = state.token;

// Expose page functions globally until we fully refactor event handlers
window.loadLicenses = loadLicenses;
window.openLicenseModal = openLicenseModal;
window.viewLicense = viewLicense;
window.deleteLicense = deleteLicense;
window.loadAdmins = loadAdmins;
window.loadDashboardStats = loadDashboardStats;
window.loadRecentActivities = loadRecentActivities;

document.addEventListener('DOMContentLoaded', () => {
  setupModalBehaviors();
  setupEventListeners();
  if (state.token) {
    showDashboard();
    fetchMeAndGateUI();
  } else {
    showLogin();
  }
});

function setupEventListeners() {
  document.getElementById('login-form')?.addEventListener('submit', handleLogin);
  document.getElementById('logout-btn')?.addEventListener('click', handleLogout);
  
  // Modal open buttons - store trigger element for focus return
  const changePwBtn = document.getElementById('change-password-btn');
  if (changePwBtn) {
    changePwBtn.addEventListener('click', () => {
      const modal = document.getElementById('change-password-modal');
      if (modal) {
        modal._triggerElement = changePwBtn;
        openModal(modal);
      }
    });
  }
  
  const createAdminBtn = document.getElementById('create-admin-btn');
  if (createAdminBtn) {
    createAdminBtn.addEventListener('click', () => {
      const modal = document.getElementById('create-admin-modal');
      if (modal) {
        modal._triggerElement = createAdminBtn;
        openModal(modal);
      }
    });
  }
  
  document.querySelectorAll('.nav-link').forEach(link => link.addEventListener('click', (e) => { e.preventDefault(); switchContent(e.target.dataset.page); }));
  
  const createLicenseBtn = document.getElementById('create-license-btn');
  if (createLicenseBtn) {
    createLicenseBtn.addEventListener('click', () => {
      const modal = document.getElementById('license-modal');
      if (modal) modal._triggerElement = createLicenseBtn;
      openLicenseModal();
    });
  }
  
  document.getElementById('create-product-btn')?.addEventListener('click', () => {
    console.log('Create product button clicked');
    console.log('window.openProductModal:', typeof window.openProductModal);
    if (window.openProductModal) {
      window.openProductModal();
    } else {
      console.error('openProductModal function not found!');
    }
  });
  document.getElementById('cleanup-devices-btn')?.addEventListener('click', () => window.openCleanupModal && window.openCleanupModal());
  document.getElementById('cleanup-confirm-btn')?.addEventListener('click', (e) => window.handleCleanupDevices && window.handleCleanupDevices(e));
  
  // License form 이벤트 리스너는 한 번만 등록 - 중복 방지
  const licenseForm = document.getElementById('license-form');
  if (licenseForm) {
    // 기존 리스너 제거
    licenseForm.replaceWith(licenseForm.cloneNode(true));
    // 새 리스너 등록
    document.getElementById('license-form')?.addEventListener('submit', handleCreateLicense);
  }
  
  // Product form 이벤트 리스너는 한 번만 등록 - 중복 방지
  const productForm = document.getElementById('product-form');
  if (productForm) {
    // 기존 리스너 제거
    productForm.replaceWith(productForm.cloneNode(true));
    // 새 리스너 등록
    document.getElementById('product-form')?.addEventListener('submit', (e) => window.handleCreateProduct && window.handleCreateProduct(e));
  }
  document.getElementById('create-admin-form')?.addEventListener('submit', handleCreateAdmin);
  
  // Modal close buttons
  document.querySelectorAll('.modal-close').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      closeModal(e.target.closest('.modal'));
    });
  });
  
  document.getElementById('change-password-form')?.addEventListener('submit', (e) => window.handleChangePassword && window.handleChangePassword(e));
  document.getElementById('search-input')?.addEventListener('input', debounce(handleSearch, 500));
  document.getElementById('status-filter')?.addEventListener('change', handleFilter);
  document.getElementById('activities-apply')?.addEventListener('click', () => loadRecentActivities());
  document.getElementById('activities-type')?.addEventListener('change', () => loadRecentActivities());
  document.getElementById('activities-action')?.addEventListener('keydown', (e) => { if (e.key === 'Enter') loadRecentActivities(); });
  document.getElementById('activities-limit')?.addEventListener('change', () => loadRecentActivities());
}

async function handleLogin(e) {
  e.preventDefault();
  const username = document.getElementById('username').value;
  const password = document.getElementById('password').value;
  const errorDiv = document.getElementById('login-error');
  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) });
    const data = await response.json();
    if (data.status === 'success') {
      state.token = data.data.token;
      window.token = state.token;
      localStorage.setItem('token', state.token);
      localStorage.setItem('username', data.data.admin.username);
      showDashboard();
      fetchMeAndGateUI();
    } else {
      errorDiv.textContent = data.message || '로그인에 실패했습니다.';
    }
  } catch (err) {
    errorDiv.textContent = '서버 연결에 실패했습니다.';
    console.error('Login error:', err);
  }
}

function handleLogout() {
  state.token = null;
  window.token = null;
  localStorage.removeItem('token');
  localStorage.removeItem('username');
  showLogin();
}

function showLogin() {
  document.getElementById('login-page').classList.add('active');
  document.getElementById('dashboard-page').classList.remove('active');
}

function showDashboard() {
  document.getElementById('login-page').classList.remove('active');
  document.getElementById('dashboard-page').classList.add('active');
  const username = localStorage.getItem('username');
  document.getElementById('user-name').textContent = username || 'Admin';
  loadDashboardStats();
  loadRecentActivities();
}

function switchContent(page) {
  document.querySelectorAll('.nav-link').forEach(link => { link.classList.toggle('active', link.dataset.page === page); });
  document.querySelectorAll('.content').forEach(c => c.classList.remove('active'));
  document.getElementById(`${page}-content`)?.classList.add('active');
  if (page === 'dashboard') { loadDashboardStats(); loadRecentActivities(); }
  else if (page === 'licenses') { loadLicenses(); }
  else if (page === 'products') { window.loadProducts && window.loadProducts(); }
  else if (page === 'admins') { loadAdmins(); }
  else if (page === 'swagger') { window.open('/swagger/index.html', '_blank'); }
}

async function fetchMeAndGateUI() {
  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/me`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const body = await res.json();
    if (res.ok && body.status === 'success') {
      state.currentRole = body.data?.role || null;
      if (state.currentRole === 'super_admin') {
        const tab = document.getElementById('admins-tab');
        if (tab) tab.style.display = '';
        document.querySelector('[data-page="admins"]')?.addEventListener('click', () => loadAdmins());
      } else {
        const cleanupBtn = document.getElementById('cleanup-devices-btn');
        if (cleanupBtn) cleanupBtn.style.display = 'none';
      }
    }
  } catch (e) { console.warn('Failed to fetch me:', e); }
}

// Load legacy SPA (app.js + products.js) for device/product/password handlers still there
import './legacy-bridge.js';
