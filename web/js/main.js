// CRITICAL: Define login handler BEFORE imports to ensure it's available globally
window.handleLoginForm = async function(event) {
    event.preventDefault();
    
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const errorEl = document.getElementById('login-error');
    
    // 에러 메시지 초기화
    if (errorEl) errorEl.textContent = '';
    
    try {
        const response = await fetch('/api/admin/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        const data = await response.json();
        
        if (response.ok && data.status === 'success') {
            // 응답에서 토큰 추출
            const token = data.data?.token || data.token;
            if (!token) {
                if (errorEl) errorEl.textContent = '토큰을 받지 못했습니다.';
                console.error('Token not received:', data);
                return;
            }
            
            localStorage.setItem('token', token);
            // 로그인 성공 시 바로 대시보드로 이동
            window.location.href = '/web/?page=dashboard';
        } else {
            const errorMsg = data.message || data.error || '알 수 없는 오류';
            if (errorEl) errorEl.textContent = '로그인 실패: ' + errorMsg;
        }
    } catch (error) {
        if (errorEl) errorEl.textContent = '오류: ' + error.message;
        console.error('Login error:', error);
    }
};

import { state } from './state.js';
import { apiFetch, API_BASE_URL } from './api.js';
import { setupModalBehaviors, openModal, closeModal, showAlert, showConfirm } from './modals.js';
import { formatDate, formatDateTime, debounce } from './utils.js';
import { renderStatusBadge } from './ui.js';
import { loadLicenses, openLicenseModal, handleCreateLicense, viewLicense, deleteLicense, handleSearch, handleFilter, handleEditLicense } from './pages/licenses.js';
import { loadAdmins, handleCreateAdmin } from './pages/admins.js';
import { loadDashboardStats, loadRecentActivities } from './pages/dashboard.js';
import { loadPolicies, openCreatePolicyModal, handleCreatePolicy, handleEditPolicy } from './pages/policies.js'; // 정책 관리 페이지
import { loadProducts, showProductModal, initProductsPage } from './pages/products.js'; // 제품 관리 페이지
import { initClientLogsPage } from './pages/client-logs.js'; // 클라이언트 로그 페이지
import './pages/account.js'; // 비밀번호 변경 (전역 핸들러 등록)
import './pages/maintenance.js'; // 디바이스 정리 (전역 핸들러 등록)
import './pages/devices.js'; // 디바이스 관리 (전역 핸들러 등록)

// Expose helpers globally for HTML onclick handlers and cross-module access
window.apiFetch = apiFetch;
window.API_BASE_URL = API_BASE_URL;
window.formatDate = formatDate;
window.formatDateTime = formatDateTime;
window.showAlert = showAlert;
window.showConfirm = showConfirm;
window.openModal = openModal;
window.closeModal = closeModal;
window.renderStatusBadge = renderStatusBadge;
window.state = state;
window.token = state.token;

// Expose page functions globally for HTML onclick handlers
window.loadLicenses = loadLicenses;
window.openLicenseModal = openLicenseModal;
window.viewLicense = viewLicense;
window.deleteLicense = deleteLicense;
window.loadAdmins = loadAdmins;
window.loadDashboardStats = loadDashboardStats;
window.loadRecentActivities = loadRecentActivities;
window.loadProducts = loadProducts;
window.showProductModal = showProductModal;
window.initProductsPage = initProductsPage;
window.handleLogout = handleLogout;

document.addEventListener('DOMContentLoaded', () => {
  console.log('📄 DOMContentLoaded event fired');
  
  state.token = localStorage.getItem('token');
  
  setupModalBehaviors();
  setupEventListeners();
  
  if (state.token) {
    console.log('✅ Token found, showing dashboard');
    showDashboard();
    fetchMeAndGateUI();
  } else {
    console.log('❌ No token, showing login');
    showLogin();
  }
});

function setupEventListeners() {
  console.log('⚙️ setupEventListeners called');
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
  
  document.querySelectorAll('.nav-link').forEach(link => link.addEventListener('click', (e) => { 
    e.preventDefault(); 
    switchContent(e.target.dataset.page); 
  }));
  
  const createLicenseBtn = document.getElementById('create-license-btn');
  if (createLicenseBtn) {
    createLicenseBtn.addEventListener('click', () => {
      const modal = document.getElementById('license-modal');
      if (modal) modal._triggerElement = createLicenseBtn;
      openLicenseModal();
    });
  }
  
  document.getElementById('create-product-btn')?.addEventListener('click', () => {
    if (window.openProductModal) {
      window.openProductModal();
    } else {
      console.error('openProductModal function not found!');
    }
  });
  
  document.getElementById('cleanup-devices-btn')?.addEventListener('click', () => {
    if (window.openCleanupModal) window.openCleanupModal();
  });
  
  document.getElementById('cleanup-confirm-btn')?.addEventListener('click', (e) => {
    if (window.handleCleanupDevices) window.handleCleanupDevices(e);
  });
  
  // License form
  const licenseForm = document.getElementById('license-form');
  if (licenseForm) {
    licenseForm.replaceWith(licenseForm.cloneNode(true));
    document.getElementById('license-form')?.addEventListener('submit', handleCreateLicense);
  }
  
  // Product form
  const productForm = document.getElementById('product-form');
  if (productForm) {
    productForm.replaceWith(productForm.cloneNode(true));
    document.getElementById('product-form')?.addEventListener('submit', (e) => {
      if (window.handleCreateProduct) window.handleCreateProduct(e);
    });
  }
  
  document.getElementById('create-admin-form')?.addEventListener('submit', handleCreateAdmin);
  
  // Policy forms
  const createPolicyForm = document.getElementById('create-policy-form');
  if (createPolicyForm) {
    createPolicyForm.replaceWith(createPolicyForm.cloneNode(true));
    document.getElementById('create-policy-form')?.addEventListener('submit', handleCreatePolicy);
  }
  
  const editPolicyForm = document.getElementById('edit-policy-form');
  if (editPolicyForm) {
    editPolicyForm.replaceWith(editPolicyForm.cloneNode(true));
    document.getElementById('edit-policy-form')?.addEventListener('submit', handleEditPolicy);
  }
  
  // License edit form
  const editLicenseForm = document.getElementById('edit-license-form');
  if (editLicenseForm) {
    editLicenseForm.replaceWith(editLicenseForm.cloneNode(true));
    document.getElementById('edit-license-form')?.addEventListener('submit', handleEditLicense);
  }
  
  // Modal close buttons
  document.querySelectorAll('.modal-close').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      closeModal(e.target.closest('.modal'));
    });
  });
  
  document.getElementById('change-password-form')?.addEventListener('submit', (e) => {
    if (window.handleChangePassword) window.handleChangePassword(e);
  });
  
  document.getElementById('search-input')?.addEventListener('input', debounce(handleSearch, 500));
  document.getElementById('status-filter')?.addEventListener('change', handleFilter);
  document.getElementById('activities-apply')?.addEventListener('click', () => loadRecentActivities());
  document.getElementById('activities-type')?.addEventListener('change', () => loadRecentActivities());
  document.getElementById('activities-action')?.addEventListener('keydown', (e) => { 
    if (e.key === 'Enter') loadRecentActivities(); 
  });
  document.getElementById('activities-limit')?.addEventListener('change', () => loadRecentActivities());

  // 관리자 테이블: 이벤트 위임
  const adminsTbody = document.getElementById('admins-tbody');
  if (adminsTbody) {
    adminsTbody.addEventListener('click', async (e) => {
      const target = e.target.closest('[data-action]');
      if (!target) return;
      e.preventDefault();
      
      const action = target.dataset.action;
      const adminId = target.dataset.adminId;
      const adminName = target.dataset.adminName;
      if (!adminId) return;

      try {
        if (action === 'reset' && window.resetAdminPassword) {
          await window.resetAdminPassword(adminId, adminName, target);
        } else if (action === 'delete' && window.deleteAdmin) {
          await window.deleteAdmin(adminId, adminName, target);
        }
      } catch (err) {
        console.error('Admin action failed:', err);
      }
    });
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
  document.querySelectorAll('.nav-link').forEach(link => { 
    link.classList.toggle('active', link.dataset.page === page); 
  });
  document.querySelectorAll('.content').forEach(c => c.classList.remove('active'));
  document.getElementById(`${page}-content`)?.classList.add('active');

  // 페이지별 로더
  if (page === 'dashboard') {
    loadDashboardStats();
    loadRecentActivities();
  } else if (page === 'licenses') {
    loadLicenses();
  } else if (page === 'admins') {
    loadAdmins();
  } else if (page === 'policies') {
    loadPolicies();
  } else if (page === 'products') {
    initProductsPage();
  } else if (page === 'client-logs') {
    initClientLogsPage();
  } else if (page === 'swagger') {
    // Swagger 페이지는 iframe으로 로드
  }
}

async function fetchMeAndGateUI() {
  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/me`, { 
      headers: { 'Authorization': `Bearer ${state.token}` } 
    });
    const body = await res.json();
    
    if (res.ok && body.status === 'success') {
      state.currentRole = body.data?.role || null;
      
      if (state.currentRole === 'super_admin') {
        const tab = document.getElementById('admins-tab');
        if (tab) tab.style.display = '';
      } else {
        const cleanupBtn = document.getElementById('cleanup-devices-btn');
        if (cleanupBtn) cleanupBtn.style.display = 'none';
      }
    }
  } catch (e) { 
    console.warn('Failed to fetch me:', e); 
  }
}
