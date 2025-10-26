// API 기본 URL
const API_BASE_URL = 'http://localhost:8080';

// 전역 상태
let token = localStorage.getItem('token');
let currentPage = 1;
let currentStatus = '';
let currentSearch = '';
let currentRole = null;
// 모달 레이어 관리 (글로벌 로딩 z-index=13000보다 항상 위)
let topZIndex = 13000;

// 글로벌 로딩 오버레이 제어
function showLoading() {
    const el = document.getElementById('global-loading');
    if (el) el.style.display = 'flex';
}
function hideLoading() {
    const el = document.getElementById('global-loading');
    if (el) el.style.display = 'none';
}

// fetch 래퍼: API 호출 자동 로딩 처리
async function apiFetch(url, options = {}) {
    const useGlobal = !options._noGlobalLoading;
    try {
        if (useGlobal) showLoading();
        const res = await fetch(url, options);
        return res;
    } finally {
        if (useGlobal) hideLoading();
    }
}

// 페이지 로드 시
document.addEventListener('DOMContentLoaded', () => {
    if (token) {
        showDashboard();
        // 현재 사용자 역할 가져오기 및 UI 게이트
        fetchMeAndGateUI();
    } else {
        showLogin();
    }
    
    // NOTE: setupEventListeners()는 main.js의 setupEventListeners()에서 호출됨 (중복 호출 방지)
    setupModalBehaviors();
    // Avoid M.AutoInit to prevent framework from hijacking our custom modals
    // If needed, selectively initialize components here (e.g., selects or tooltips)
    // Example:
    // if (window.M && M.FormSelect) { M.FormSelect.init(document.querySelectorAll('select')); }
});

// 이벤트 리스너 설정
function setupEventListeners() {
    // NOTE: 모든 이벤트 리스너는 main.js의 setupEventListeners()에서 등록됨
    // 이 함수는 app.js의 레거시 코드 호환성을 위해 유지됨
}

// 모달 공통 동작 (오버레이 클릭, ESC 닫기, body 스크롤 잠금)
function setupModalBehaviors() {
    // 오버레이 클릭 시 닫기 (콘텐츠 클릭은 제외)
    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) closeModal(modal);
        });
    });

    // ESC 키로 닫기
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const opened = document.querySelector('.modal.active');
            if (opened) closeModal(opened);
        }
    });
}

function openModal(modal) {
    // 최상위로 올리기: 모달 래퍼와 콘텐츠 z-index를 점증적으로 부여
    topZIndex += 2; // 래퍼와 콘텐츠용 2단계 확보
    modal.style.zIndex = String(topZIndex);
    const contentEl = modal.querySelector('.modal-content');
    if (contentEl) contentEl.style.zIndex = String(topZIndex + 1);

    modal.classList.add('active');
    document.body.classList.add('modal-open');
    // 포커스 트랩 시작
    const focusable = modal.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (first) first.focus();
    modal._trap = (e) => {
        if (e.key === 'Tab') {
            if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
            else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
        }
        if (e.key === 'Escape') {
            closeModal(modal);
        }
    };
    modal.addEventListener('keydown', modal._trap);
}

function closeModal(modal) {
    modal.classList.remove('active');
    // 약간의 지연 후 스크롤 잠금 해제 (여러 모달 중첩 고려)
    setTimeout(() => {
        if (!document.querySelector('.modal.active')) {
            document.body.classList.remove('modal-open');
        }
    }, 200);
    if (modal._trap) {
        modal.removeEventListener('keydown', modal._trap);
        delete modal._trap;
    }
}

// 비밀번호 변경 처리
async function handleChangePassword(e) {
    e.preventDefault();
    const oldPassword = document.getElementById('current_password').value;
    const newPassword = document.getElementById('new_password').value;
    const newPasswordConfirm = document.getElementById('new_password_confirm').value;

    if (newPassword !== newPasswordConfirm) {
        await showAlert('새 비밀번호 확인이 일치하지 않습니다.', '비밀번호 변경');
        return;
    }
    if (newPassword.length < 8) {
        await showAlert('새 비밀번호는 8자 이상이어야 합니다.', '비밀번호 변경');
        return;
    }

    try {
        // 현재 저장된 토큰 확인
        const currentToken = localStorage.getItem('token');
        if (!currentToken) {
            await showAlert('인증 토큰이 없습니다. 다시 로그인 해주세요.', '비밀번호 변경');
            handleLogout();
            return;
        }

        const response = await apiFetch(`${API_BASE_URL}/api/admin/change-password`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${currentToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
        });

        const result = await response.json();
        
        // 먼저 모달 닫기
        const modal = document.getElementById('change-password-modal');
        if (modal) {
            modal.classList.remove('active');
        }
        e.target.reset();
        
        if (response.ok && result.status === 'success') {
            setTimeout(async () => {
                await showAlert('비밀번호가 변경되었습니다. 다시 로그인 해주세요.', '비밀번호 변경 완료');
                // 보안을 위해 토큰 초기화 및 로그아웃
                handleLogout();
            }, 300);
        } else {
            setTimeout(() => {
                showAlert(result.message || '비밀번호 변경에 실패했습니다.', '비밀번호 변경 실패');
            }, 300);
        }
    } catch (error) {
        console.error('Failed to change password:', error);
        // 에러 시에도 모달 닫기
        const modal = document.getElementById('change-password-modal');
        if (modal) {
            modal.classList.remove('active');
        }
        e.target.reset();
        
        setTimeout(() => {
            showAlert('서버 오류가 발생했습니다.', '비밀번호 변경 실패');
        }, 300);
    }
}

// 로그인 처리 - 이제 main.js에서 처리됨 (혼동 방지)
/*
async function handleLogin(e) {
    e.preventDefault();
    
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const errorDiv = document.getElementById('login-error');
    
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            token = data.data.token;
            localStorage.setItem('token', token);
            localStorage.setItem('username', data.data.admin.username);
            showDashboard();
            // 로그인 직후 역할 기반 UI 갱신
            fetchMeAndGateUI();
        } else {
            errorDiv.textContent = data.message || '로그인에 실패했습니다.';
        }
    } catch (error) {
        errorDiv.textContent = '서버 연결에 실패했습니다.';
        console.error('Login error:', error);
    }
}
*/

// 로그아웃 처리 - 이제 main.js에서 처리됨
/*
function handleLogout() {
    token = null;
    localStorage.removeItem('token');
    localStorage.removeItem('username');
    showLogin();
}
*/

// 로그인 페이지 표시
function showLogin() {
    document.getElementById('login-page').classList.add('active');
    document.getElementById('dashboard-page').classList.remove('active');
}

// 대시보드 표시
function showDashboard() {
    document.getElementById('login-page').classList.remove('active');
    document.getElementById('dashboard-page').classList.add('active');
    
    // 사용자 이름 표시
    const username = localStorage.getItem('username');
    document.getElementById('user-name').textContent = username || 'Admin';
    
    // 대시보드 데이터 로드
    loadDashboardStats();
    loadRecentActivities();
}

// 현재 사용자 조회 및 역할 기반 UI 게이트
async function fetchMeAndGateUI() {
    try {
        const res = await apiFetch(`${API_BASE_URL}/api/admin/me`, { headers: { 'Authorization': `Bearer ${token}` } });
        const body = await res.json();
        if (res.ok && body.status === 'success') {
            currentRole = body.data?.role || null;
            // 슈퍼 관리자면 관리자 탭/버튼 노출
            if (currentRole === 'super_admin') {
                const tab = document.getElementById('admins-tab');
                if (tab) tab.style.display = '';
                // 기본 로드: 관리자 탭이 열릴 때 목록 불러오기
                document.querySelector('[data-page="admins"]').addEventListener('click', () => loadAdmins());
            } else {
                // 위험 작업 버튼 감춤 (정리)
                const cleanupBtn = document.getElementById('cleanup-devices-btn');
                if (cleanupBtn) cleanupBtn.style.display = 'none';
            }
        }
    } catch (e) {
        console.warn('Failed to fetch me:', e);
    }
}

// 컨텐츠 전환
function switchContent(page) {
    // 네비게이션 활성화
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
        if (link.dataset.page === page) {
            link.classList.add('active');
        }
    });
    
    // 컨텐츠 전환
    document.querySelectorAll('.content').forEach(content => {
        content.classList.remove('active');
    });
    
    const targetContent = document.getElementById(`${page}-content`);
    if (targetContent) {
        targetContent.classList.add('active');
    }
    
    // 페이지별 데이터 로드
    if (page === 'dashboard') {
        loadDashboardStats();
        loadRecentActivities();
    } else if (page === 'licenses') {
        loadLicenses();
        } else if (page === 'products') {
            loadProducts();
        } else if (page === 'admins') {
            loadAdmins();
        } else if (page === 'swagger') {
            window.open('/swagger/index.html', '_blank');
    }
}

// 대시보드 통계 로드
async function loadDashboardStats() {
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/dashboard/stats`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            const stats = data.data;
            document.getElementById('total-licenses').textContent = stats.total_licenses || 0;
            document.getElementById('active-licenses').textContent = stats.active_licenses || 0;
            document.getElementById('expired-licenses').textContent = stats.expired_licenses || 0;
            document.getElementById('total-devices').textContent = stats.total_active_devices || 0;
        }
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// 관리자 목록 로드 (슈퍼 전용)
async function loadAdmins() {
    try {
        const res = await apiFetch(`${API_BASE_URL}/api/admin/admins`, { headers: { 'Authorization': `Bearer ${token}` } });
        const body = await res.json();
        const tbody = document.getElementById('admins-tbody');
        if (!tbody) return;
        if (res.ok && body.status === 'success') {
            const admins = body.data || [];
            if (admins.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" class="text-center">관리자가 없습니다.</td></tr>';
            } else {
                tbody.innerHTML = admins.map(a => `
                    <tr>
                        <td>${escapeHtml(a.username)} <small class="mono" style="color:#777;">(${escapeHtml(a.id)})</small></td>
                        <td>${escapeHtml(a.email || '-')}</td>
                        <td>
                            <span class="role-badge ${a.role === 'super_admin' ? 'super' : 'admin'}">
                                <span class="icon">${a.role === 'super_admin' ? '⭐' : '👤'}</span>
                                ${a.role === 'super_admin' ? 'Super Admin' : 'Admin'}
                            </span>
                        </td>
                        <td>${formatDateTime(a.created_at)}</td>
                    </tr>
                `).join('');
            }
        } else {
            tbody.innerHTML = `<tr><td colspan="4" class="text-center">불러오기에 실패했습니다: ${escapeHtml(body.message || '')}</td></tr>`;
        }
    } catch (e) {
        console.error('Failed to load admins:', e);
        const tbody = document.getElementById('admins-tbody');
        if (tbody) tbody.innerHTML = '<tr><td colspan="4" class="text-center">서버 오류</td></tr>';
    }
}

// 서브 관리자 생성 (슈퍼 전용)
async function handleCreateAdmin(e) {
    e.preventDefault();
    const username = document.getElementById('admin_username').value.trim();
    const email = document.getElementById('admin_email').value.trim();
    const password = document.getElementById('admin_password').value;
    if (!username || !email || !password) return;

    const submitBtn = e.target.querySelector('button[type="submit"]');
    const originalBtnText = submitBtn ? submitBtn.textContent : '';
    if (submitBtn) { submitBtn.disabled = true; submitBtn.textContent = '생성 중...'; }

    try {

        const res = await apiFetch(`${API_BASE_URL}/api/admin/admins/create`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, email, password }),
            _noGlobalLoading: true
        });
        const body = await res.json();
        if (res.ok && body.status === 'success') {
            // 먼저 생성 모달을 닫은 다음 알림을 띄워 겹침/가림 문제를 방지
            const createAdminModal = document.getElementById('create-admin-modal');
            if (createAdminModal) closeModal(createAdminModal);
            e.target.reset();
            await showAlert('서브 관리자가 생성되었습니다.', '관리자 생성');
            loadAdmins();
            loadRecentActivities();
        } else {
            await showAlert(body.message || '생성에 실패했습니다.', '관리자 생성 실패');
        }
    } catch (err) {
        console.error('Failed to create admin:', err);
        await showAlert('서버 오류가 발생했습니다.', '관리자 생성 실패');
    } finally {
        if (submitBtn) { submitBtn.disabled = false; submitBtn.textContent = originalBtnText; }
    }
}

// 최근 활동 로드
async function loadRecentActivities() {
    try {
        const type = document.getElementById('activities-type')?.value || '';
        const action = document.getElementById('activities-action')?.value.trim() || '';
        const limit = document.getElementById('activities-limit')?.value || '20';
        const params = new URLSearchParams();
        if (type) params.set('type', type);
        if (action) params.set('action', action);
        if (limit) params.set('limit', limit);
        const qs = params.toString();
        const response = await apiFetch(`${API_BASE_URL}/api/admin/dashboard/activities${qs ? `?${qs}` : ''}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            const activities = data.data;
            const container = document.getElementById('recent-activities');

            const actionIcons = {
                'activated': '🟢 활성화',
                'validated': '✅ 검증',
                'deactivated': '🔴 비활성화',
                'reactivated': '🔄 재활성화',
                'admin:login': '👤 로그인',
                'admin:change_password': '🔐 비밀번호 변경',
                'admin:create_admin': '👥 관리자 생성',
                'admin:create_product': '📦 제품 생성',
                'admin:update_product': '🛠️ 제품 수정',
                'admin:delete_product': '🗑️ 제품 삭제',
                'admin:create_license': '🎫 라이선스 생성',
                'admin:update_license': '🛠️ 라이선스 수정',
                'admin:deactivate_device': '👮 디바이스 비활성화',
                'admin:reactivate_device': '👮 디바이스 재활성화',
                'admin:cleanup_devices': '🧹 디바이스 정리'
            };

            if (activities && activities.length > 0) {
                container.innerHTML = activities.map(a => {
                    if (a.type === 'admin') {
                        const label = actionIcons[`admin:${a.action}`] || `👤 ${a.action}`;
                        const details = a.details ? `<div style="color:#6b7280; font-size: 0.9em; margin-top: 4px;">${escapeHtml(a.details)}</div>` : '';
                        return `
                        <div class="activity-item">
                            <div>
                                <div><strong>관리자</strong> · ${escapeHtml(a.admin_username || '-')}</div>
                                <div style="margin-top:2px; color:#374151;">${label}</div>
                                ${details}
                            </div>
                            <div><small>${formatDateTime(a.created_at)}</small></div>
                        </div>`;
                    } else {
                        const actionLabel = actionIcons[a.action] || `📝 ${a.action}`;
                        const product = a.product_name ? ` <span class="badge">${escapeHtml(a.product_name)}</span>` : '';
                        const fp = a.fingerprint ? `<code style="font-size: 0.8em;">${escapeHtml(a.fingerprint)}</code>` : '';
                        const details = a.details ? `<div style="color:#6b7280; font-size: 0.9em; margin-top: 4px;">${escapeHtml(a.details)}</div>` : '';
                        return `
                        <div class="activity-item">
                            <div>
                                <div><strong>${escapeHtml(a.customer_name || '-')}</strong> - ${escapeHtml(a.license_key || '-')}${product}</div>
                                <div style="margin-top:2px; color:#374151;">
                                    ${actionLabel} · ${escapeHtml(a.device_name || '-')}
                                    ${fp}
                                </div>
                                ${details}
                            </div>
                            <div>
                                <small>${formatDateTime(a.created_at)}</small>
                            </div>
                        </div>`;
                    }
                }).join('');
            } else {
                container.innerHTML = '<p class="loading">활동 내역이 없습니다.</p>';
            }
        }
    } catch (error) {
        console.error('Failed to load activities:', error);
    }
}

// 라이선스 목록 로드
async function loadLicenses(page = 1) {
    try {
        let url = `${API_BASE_URL}/api/admin/licenses?page=${page}&page_size=10`;
        
        if (currentStatus) {
            url += `&status=${currentStatus}`;
        }
        
        if (currentSearch) {
            url += `&search=${encodeURIComponent(currentSearch)}`;
        }
        
        const response = await apiFetch(url, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            renderLicensesTable(data.data);
            renderPagination(data.meta);
        }
    } catch (error) {
        console.error('Failed to load licenses:', error);
    }
}

// 라이선스 테이블 렌더링
function renderLicensesTable(licenses) {
    const tbody = document.getElementById('licenses-tbody');
    
    if (!licenses || licenses.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="loading">라이선스가 없습니다.</td></tr>';
        return;
    }
    
    tbody.innerHTML = licenses.map(license => `
        <tr>
            <td><code>${license.license_key}</code></td>
            <td>${license.product_name}</td>
            <td>${license.customer_name}</td>
            <td>${license.max_devices}</td>
            <td>${formatDate(license.expires_at)}</td>
            <td>${renderStatusBadge(license.status)}</td>
            <td>
                <button class="btn btn-sm" onclick="viewLicense('${license.id}')">상세</button>
                <button class="btn btn-sm btn-danger" onclick="deleteLicense('${license.id}')">삭제</button>
            </td>
        </tr>
    `).join('');
}

// 페이지네이션 렌더링
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

// 상태 뱃지 렌더링
function renderStatusBadge(status) {
    const badges = {
        'active': '<span class="status-badge status-active"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>Active</span>',
        'expired': '<span class="status-badge status-expired"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>Expired</span>',
        'revoked': '<span class="status-badge status-inactive"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>Inactive</span>'
    };
    
    return badges[status] || `<span class="status-badge"><svg class="status-icon" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>${status}</span>`;
}

// 라이선스 생성 모달 열기
function openLicenseModal() {
    // 1년 후 날짜 설정
    const nextYear = new Date();
    nextYear.setFullYear(nextYear.getFullYear() + 1);
    document.getElementById('expires_at').value = nextYear.toISOString().split('T')[0];

    // 만료일 min 오늘로 제한
    const today = new Date();
    document.getElementById('expires_at').setAttribute('min', today.toISOString().split('T')[0]);

    // 제품 셀렉트 로드
    populateProductDropdown();
    
    openModal(document.getElementById('license-modal'));
}

// 라이선스 생성 처리
async function handleCreateLicense(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const data = {
        product_id: formData.get('product_id') || '',
        customer_name: formData.get('customer_name'),
        customer_email: formData.get('customer_email'),
        max_devices: parseInt(formData.get('max_devices')),
        expires_at: new Date(formData.get('expires_at')).toISOString(),
        notes: formData.get('notes')
    };

    // hidden product_id가 없을 경우 select 값으로 대체
    if (!data.product_id) {
        const sel = document.getElementById('product_select');
        if (sel && sel.value) data.product_id = sel.value;
    }

    // 과거 날짜 방지 클라이언트 검증
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
        const response = await apiFetch(`${API_BASE_URL}/api/admin/licenses`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data),
            _noGlobalLoading: true
        });
        
        const result = await response.json();
        
        if (result.status === 'success') {
            // 모달 먼저 닫고 메시지 띄우기 (겹침 방지)
            const licenseModal = document.getElementById('license-modal');
            if (licenseModal) closeModal(licenseModal);
            await showAlert(`라이선스가 생성되었습니다!\n라이선스 키: ${result.data.license_key}`, '라이선스 생성');
            e.target.reset();
            loadLicenses();
            loadDashboardStats();
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

// 제품 드롭다운 채우기 및 연동
async function populateProductDropdown() {
    const select = document.getElementById('product_select');
    if (!select) return;

    // 초기화
    select.innerHTML = '';

    try {
        const res = await apiFetch(`${API_BASE_URL}/api/admin/products?status=active`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) throw new Error('Failed to load products');
        const body = await res.json();
        const products = body.data || [];

        products.forEach(p => {
            const opt = document.createElement('option');
            opt.value = p.id; // product_id
            opt.textContent = `${p.name}`;
            select.appendChild(opt);
        });

        // 선택 시 hidden product_id 저장 (있을 때만)
        let hiddenId = document.getElementById('product_id_hidden');
        select.onchange = () => {
            if (!hiddenId) return;
            const selected = select.selectedOptions[0];
            hiddenId.value = selected && selected.value ? selected.value : '';
        };

        // 제품이 있으면 첫 번째 제품을 기본 선택
        if (products.length > 0) select.value = products[0].id;
        select.required = true;
        select.onchange();
    } catch (err) {
        console.error('Failed to populate product dropdown:', err);
    }
}

// 라이선스 상세 보기
async function viewLicense(id) {
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/licenses/?id=${id}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            const license = data.data;
            const content = `
                <div class="detail-group">
                    <div class="detail-label">라이선스 키</div>
                    <div class="detail-value"><code>${license.license_key}</code></div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">제품명</div>
                    <div class="detail-value">${license.product_name}</div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">고객 정보</div>
                    <div class="detail-value">${license.customer_name}<br>${license.customer_email}</div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">최대 디바이스</div>
                    <div class="detail-value">${license.max_devices}</div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">만료일</div>
                    <div class="detail-value">${formatDate(license.expires_at)}</div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">상태</div>
                    <div class="detail-value">${renderStatusBadge(license.status)}</div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">메모</div>
                    <div class="detail-value">${license.notes || '-'}</div>
                </div>
                <div class="detail-group">
                    <div class="detail-label">등록된 디바이스</div>
                    <div class="detail-value">
                        <div id="license-devices" class="device-list loading">불러오는 중...</div>
                    </div>
                </div>
            `;
            
                document.getElementById('license-detail-content').innerHTML = content;
                openModal(document.getElementById('license-detail-modal'));

            // 디바이스 목록 로드
            try {
                const res = await apiFetch(`${API_BASE_URL}/api/admin/licenses/devices?id=${id}`, {
                    headers: { 'Authorization': `Bearer ${token}` }
                });
                const body = await res.json();
                const container = document.getElementById('license-devices');
                if (res.ok && body.status === 'success') {
                    const devices = body.data || [];
                    container.classList.remove('loading');
                    if (devices.length === 0) {
                        container.textContent = '등록된 디바이스가 없습니다.';
                    } else {
                        container.innerHTML = devices.map(renderDeviceCard).join('');
                    }
                } else {
                    container.classList.remove('loading');
                    container.textContent = body.message || '디바이스 정보를 불러오지 못했습니다.';
                }
            } catch (e) {
                const container = document.getElementById('license-devices');
                if (container) {
                    container.classList.remove('loading');
                    container.textContent = '디바이스 정보를 불러오지 못했습니다.';
                }
            }
        }
    } catch (error) {
        console.error('Failed to load license:', error);
    }
}

// 라이선스 삭제
async function deleteLicense(id) {
    const ok = await showConfirm('정말로 이 라이선스를 삭제하시겠습니까?', '라이선스 삭제');
    if (!ok) {
        return;
    }
    
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/licenses/?id=${id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            await showAlert('라이선스가 삭제되었습니다.', '라이선스 삭제');
            loadLicenses();
            loadDashboardStats();
        } else {
            await showAlert('삭제 실패: ' + data.message, '라이선스 삭제');
        }
    } catch (error) {
        await showAlert('서버 오류가 발생했습니다.', '라이선스 삭제');
        console.error('Failed to delete license:', error);
    }
}

// 검색 처리
function handleSearch(e) {
    currentSearch = e.target.value;
    currentPage = 1;
    loadLicenses();
}

// 필터 처리
function handleFilter(e) {
    currentStatus = e.target.value;
    currentPage = 1;
                    closeModal(document.getElementById('license-modal'));
}

// 유틸리티 함수들

// 날짜 포맷팅
function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleDateString('ko-KR', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
    });
}

// 날짜+시간 포맷팅
function formatDateTime(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString('ko-KR', {
        year: '2-digit',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// 디바운스
function debounce(func, wait) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}

// 공통 다이얼로그
function openDialog({ title = '알림', message = '', showCancel = false }) {
    const modal = document.getElementById('dialog-modal');
    const titleEl = document.getElementById('dialog-title');
    const msgEl = document.getElementById('dialog-message');
    const btnCancel = document.getElementById('dialog-cancel');
    const btnOk = document.getElementById('dialog-ok');

    titleEl.textContent = title;
    msgEl.textContent = message;
    btnCancel.style.display = showCancel ? 'inline-flex' : 'none';
    // 다이얼로그는 항상 최상단 보장
    topZIndex = Math.max(topZIndex, 13000) + 4;
    modal.style.zIndex = String(topZIndex);
    const contentEl = modal.querySelector('.modal-content');
    if (contentEl) contentEl.style.zIndex = String(topZIndex + 1);
    openModal(modal);

    return new Promise((resolve) => {
        const close = (result) => {
            closeModal(modal);
            btnOk.onclick = null;
            btnCancel.onclick = null;
            resolve(result);
        };
        btnOk.onclick = () => close(true);
        btnCancel.onclick = () => close(false);
        modal.querySelector('.modal-close').onclick = () => close(false);
    });
}

async function showAlert(message, title = '알림') {
    await openDialog({ title, message, showCancel: false });
}

async function showConfirm(message, title = '확인') {
    return await openDialog({ title, message, showCancel: true });
}

// 디바이스 카드 렌더링
function renderDeviceCard(d) {
    const info = safeParseJSON(d.device_info);
    const statusBadge = renderDeviceStatusBadge(d.status);
    const validationWarning = getValidationWarning(d.last_validated_at);
    const isActive = d.status === 'active';
    // 현재 라이선스 ID는 전역이 아니므로 data attribute로 저장
    const licenseId = d.license_id;

    return `
    <div class="device-card ${isActive ? '' : 'inactive'} card">
        <div class="device-card-header">
            <div class="device-name">💻 <strong>${escapeHtml(d.device_name || '이름 없음')}</strong></div>
            <div class="device-actions">
                ${statusBadge}
                ${isActive 
                    ? `<button class="btn btn-sm btn-danger" onclick="deactivateDevice('${d.id}', '${escapeHtml(d.device_name)}', '${licenseId}')">비활성화</button>` 
                    : `<button class="btn btn-sm btn-success" onclick="reactivateDevice('${d.id}', '${escapeHtml(d.device_name)}', '${licenseId}')">재활성화</button>`
                }
            </div>
        </div>
        <div class="device-card-body">
            <div class="kv-list">
                <div class="kv-row"><span class="kv-key">Client ID</span><span class="kv-val mono">${escapeHtml(info.client_id || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">Hostname</span><span class="kv-val">${escapeHtml(info.hostname || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">Machine ID</span><span class="kv-val mono">${escapeHtml(info.machine_id || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">CPU ID</span><span class="kv-val mono">${escapeHtml(info.cpu_id || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">Motherboard SN</span><span class="kv-val mono">${escapeHtml(info.motherboard_sn || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">MAC Address</span><span class="kv-val mono">${escapeHtml(info.mac_address || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">Disk Serial</span><span class="kv-val mono">${escapeHtml(info.disk_serial || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">OS</span><span class="kv-val">${escapeHtml(info.os || '-')}</span></div>
                <div class="kv-row"><span class="kv-key">OS Version</span><span class="kv-val">${escapeHtml(info.os_version || '-')}</span></div>
            </div>
        </div>
        <div class="device-card-footer">
            <div class="fingerprint-wrapper">
                <small>🔑 <code id="fp-${d.id}">${escapeHtml(d.device_fingerprint)}</code></small>
                <button class="btn-copy" onclick="copyToClipboard('fp-${d.id}')" title="복사">📋</button>
            </div>
            <small>📅 등록: ${formatDate(d.activated_at)}</small>
            <small class="${validationWarning.class}">✅ 검증: ${formatDate(d.last_validated_at)} ${validationWarning.text}</small>
            <div style="margin-top: 8px;">
                <button class="btn btn-sm" onclick="toggleActivityLogs('${d.id}')">📋 활동 로그</button>
            </div>
            <div id="activity-logs-${d.id}" class="activity-logs" style="display: none;"></div>
        </div>
    </div>`;
}

function renderDeviceStatusBadge(status) {
    const map = {
        'active': '<span class="new badge green" data-badge-caption="">활성</span>',
        'deactivated': '<span class="new badge red" data-badge-caption="">비활성</span>'
    };
    return map[status] || `<span class="new badge grey" data-badge-caption="">${escapeHtml(status || '-') }</span>`;
}

function safeParseJSON(str) {
    try {
        if (!str) return {};
        return JSON.parse(str);
    } catch (_) { return {}; }
}

function escapeHtml(s) {
    return String(s)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

// 검증 경고 판단 (30일 이상 미검증 시 경고)
function getValidationWarning(lastValidated) {
    if (!lastValidated) return { class: '', text: '' };
    
    const now = new Date();
    const validated = new Date(lastValidated);
    const daysDiff = Math.floor((now - validated) / (1000 * 60 * 60 * 24));
    
    if (daysDiff > 30) {
        return { class: 'validation-warning', text: `(${daysDiff}일 전)` };
    } else if (daysDiff > 7) {
        return { class: 'validation-old', text: `(${daysDiff}일 전)` };
    }
    return { class: '', text: '' };
}

// 클립보드 복사
async function copyToClipboard(elementId) {
    const el = document.getElementById(elementId);
    if (!el) return;
    
    const text = el.textContent;
    try {
        await navigator.clipboard.writeText(text);
        // 복사 성공 피드백
        const originalText = el.parentElement.innerHTML;
        el.parentElement.innerHTML = '✅ 복사됨!';
        setTimeout(() => {
            el.parentElement.innerHTML = originalText;
        }, 1500);
    } catch (err) {
        console.error('Failed to copy:', err);
        await showAlert('클립보드 복사에 실패했습니다.', '복사 실패');
    }
}

// 디바이스 비활성화
async function deactivateDevice(deviceId, deviceName, licenseId) {
    const ok = await showConfirm(
        `"${deviceName}" 디바이스를 비활성화하시겠습니까?\n\n비활성화하면 이 디바이스에서 더 이상 라이선스를 사용할 수 없습니다.`,
        '디바이스 비활성화'
    );
    
    if (!ok) return;
    
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/deactivate`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ device_id: deviceId })
        });
        
        const result = await response.json();
        
        if (response.ok && result.status === 'success') {
            await showAlert('디바이스가 비활성화되었습니다.\n\n해당 디바이스에서는 더 이상 라이선스 검증이 실패하며, 최대 디바이스 슬롯 하나가 해제되어 새로운 디바이스를 등록할 수 있습니다.', '디바이스 비활성화');
            // 디바이스 목록만 부분 리로드
            await reloadDeviceList(licenseId);
        } else {
            await showAlert(result.message || '디바이스 비활성화에 실패했습니다.', '디바이스 비활성화');
        }
    } catch (error) {
        console.error('Failed to deactivate device:', error);
        await showAlert('서버 오류가 발생했습니다.', '디바이스 비활성화');
    }
}

// 디바이스 재활성화
async function reactivateDevice(deviceId, deviceName, licenseId) {
    const ok = await showConfirm(
        `"${deviceName}" 디바이스를 재활성화하시겠습니까?\n\n재활성화하면 이 디바이스에서 다시 라이선스를 사용할 수 있습니다. (디바이스 슬롯이 남아있어야 합니다)`,
        '디바이스 재활성화'
    );
    
    if (!ok) return;
    
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/reactivate`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ device_id: deviceId })
        });
        
        const result = await response.json();
        
        if (response.ok && result.status === 'success') {
            await showAlert('디바이스가 재활성화되었습니다.\n\n해당 디바이스에서 라이선스 검증이 다시 정상적으로 작동합니다.', '디바이스 재활성화');
            // 디바이스 목록만 부분 리로드
            await reloadDeviceList(licenseId);
        } else {
            await showAlert(result.message || '디바이스 재활성화에 실패했습니다.', '디바이스 재활성화');
        }
    } catch (error) {
        console.error('Failed to reactivate device:', error);
        await showAlert('서버 오류가 발생했습니다.', '디바이스 재활성화');
    }
}

// 디바이스 삭제 (완전 삭제) - 제거됨, 대신 cleanup 기능 사용
/*
async function deleteDevice(deviceId, deviceName, licenseId) {
    const ok = await showConfirm(
        `디바이스 "${deviceName}"를 데이터베이스에서 완전히 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.`,
        '디바이스 삭제'
    );
    
    if (!ok) return;
    
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/delete`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ device_id: deviceId })
        });
        
        const result = await response.json();
        
        if (response.ok && result.status === 'success') {
            await showAlert('디바이스가 삭제되었습니다.', '디바이스 삭제');
            await reloadDeviceList(licenseId);
        } else {
            await showAlert(result.message || '디바이스 삭제에 실패했습니다.', '디바이스 삭제');
        }
    } catch (error) {
        console.error('Failed to delete device:', error);
        await showAlert('서버 오류가 발생했습니다.', '디바이스 삭제');
    }
}
*/

// 활동 로그 토글
async function toggleActivityLogs(deviceId) {
    const logsContainer = document.getElementById(`activity-logs-${deviceId}`);
    if (!logsContainer) return;
    
    // 이미 표시 중이면 숨기기
    if (logsContainer.style.display === 'block') {
        logsContainer.style.display = 'none';
        return;
    }
    
    // 로딩 표시
    logsContainer.style.display = 'block';
    logsContainer.innerHTML = '<div style="text-align: center; padding: 10px;">로딩 중...</div>';
    
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/logs?device_id=${deviceId}`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const result = await response.json();
        
        if (response.ok && result.status === 'success') {
            const logs = result.data || [];
            if (logs.length === 0) {
                logsContainer.innerHTML = '<div style="padding: 10px; color: #999;">활동 로그가 없습니다.</div>';
            } else {
                logsContainer.innerHTML = renderActivityLogs(logs);
            }
        } else {
            logsContainer.innerHTML = '<div style="padding: 10px; color: #f44336;">로그를 불러오지 못했습니다.</div>';
        }
    } catch (error) {
        console.error('Failed to load activity logs:', error);
        logsContainer.innerHTML = '<div style="padding: 10px; color: #f44336;">서버 오류가 발생했습니다.</div>';
    }
}

// 활동 로그 렌더링
function renderActivityLogs(logs) {
    const actionIcons = {
        'activated': '🟢',
        'validated': '✅',
        'deactivated': '🔴',
        'reactivated': '🔄'
    };
    
    const actionNames = {
        'activated': '활성화',
        'validated': '검증',
        'deactivated': '비활성화',
        'reactivated': '재활성화'
    };
    
    const items = logs.map(log => {
        const icon = actionIcons[log.action] || '📝';
        const actionName = actionNames[log.action] || log.action || '알 수 없음';
        const details = log.details ? `<div style="font-size: 0.85em; color: #666; margin-top: 4px;">${escapeHtml(log.details)}</div>` : '';
        
        return `
        <div class="activity-log-item">
            <div style="display: flex; align-items: center; gap: 8px;">
                <span>${icon}</span>
                <strong>${actionName}</strong>
                <span style="color: #999; font-size: 0.9em;">${formatDateTime(log.created_at)}</span>
            </div>
            ${details}
        </div>`;
    }).join('');
    
    return `<div class="activity-log-list">${items}</div>`;
}

// 디바이스 목록 부분 리로드
async function reloadDeviceList(licenseId) {
    try {
        // 디바이스 목록 갱신
        const res = await apiFetch(`${API_BASE_URL}/api/admin/licenses/devices?id=${licenseId}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const body = await res.json();
        const container = document.getElementById('license-devices');
        if (res.ok && body.status === 'success') {
            const devices = body.data || [];
            container.classList.remove('loading');
            if (devices.length === 0) {
                container.textContent = '등록된 디바이스가 없습니다.';
            } else {
                container.innerHTML = devices.map(renderDeviceCard).join('');
            }
        } else {
            container.classList.remove('loading');
            container.textContent = body.message || '디바이스 정보를 불러오지 못했습니다.';
        }
        
        // 상세 창의 디바이스 슬롯 정보도 갱신
        const licenseRes = await apiFetch(`${API_BASE_URL}/api/admin/licenses/?id=${licenseId}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const licenseData = await licenseRes.json();
        if (licenseRes.ok && licenseData.status === 'success') {
            const license = licenseData.data;
            const activeDevices = license.active_devices || 0;
            const remainingDevices = license.max_devices - activeDevices;
            const deviceUsage = `${remainingDevices}/${license.max_devices}`;
            
            // 상세 창에서 디바이스 슬롯 값 찾아서 업데이트
            const detailGroups = document.querySelectorAll('.detail-group');
            detailGroups.forEach(group => {
                const label = group.querySelector('.detail-label');
                if (label && label.textContent === '디바이스 슬롯') {
                    const valueEl = group.querySelector('.detail-value');
                    if (valueEl) {
                        valueEl.textContent = deviceUsage;
                    }
                }
            });
        }
        
        // 라이선스 목록도 갱신 (디바이스 수 업데이트)
        if (window.loadLicenses) {
            window.loadLicenses(currentPage);
        }
    } catch (e) {
        console.error('Failed to reload device list:', e);
    }
}


// 디바이스 정리 모달 열기
function openCleanupModal() {
    const modal = document.getElementById('cleanup-modal');
    document.getElementById('cleanup-days').value = '90';
    const preview = document.getElementById('cleanup-preview');
    if (preview) preview.style.display = 'none';
    openModal(modal);
}

// 디바이스 정리 실행
async function handleCleanupDevices(e) {
    const days = parseInt(document.getElementById('cleanup-days').value);

    if (isNaN(days) || days < 0) {
        await showAlert('0 이상의 유효한 일수를 입력해주세요.', '입력 오류');
        return;
    }

    const message = days === 0 
        ? `모든 비활성 디바이스를 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.`
        : `비활성화된 지 ${days}일이 지난 디바이스를 모두 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.`;

    // 먼저 cleanup 모달 닫기 (confirm 모달과 충돌 방지)
    const cleanupModal = document.getElementById('cleanup-modal');
    if (cleanupModal) {
        closeModal(cleanupModal);
    }

    // 모달 닫힌 후 confirm 표시
    setTimeout(async () => {
        const ok = await showConfirm(message, '디바이스 정리');

        if (!ok) {
            // 취소 시 모달 다시 열기
            if (cleanupModal) {
                openModal(cleanupModal);
            }
            return;
        }

        const submitBtn = e && e.target ? e.target : document.getElementById('cleanup-confirm-btn');
        const originalBtnText = submitBtn ? submitBtn.textContent : '';
        if (submitBtn) { submitBtn.disabled = true; submitBtn.textContent = '정리 중...'; }

        try {
            const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/cleanup`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ days }),
                _noGlobalLoading: true
            });

            const result = await response.json();

            if (response.ok && result.status === 'success') {
                const count = result.data ? result.data.deleted_count : 0;
                showAlert(`${count}개의 디바이스가 삭제되었습니다.`, '디바이스 정리 완료');
                // 대시보드 통계/활동 새로고침
                if (document.getElementById('dashboard-content').classList.contains('active')) {
                    loadDashboardStats();
                    loadRecentActivities();
                }
            } else {
                showAlert(result.message || '디바이스 정리에 실패했습니다.', '디바이스 정리 실패');
            }
        } catch (error) {
            console.error('Failed to cleanup devices:', error);
            showAlert('서버 오류가 발생했습니다.', '디바이스 정리 실패');
        } finally {
            if (submitBtn) { submitBtn.disabled = false; submitBtn.textContent = originalBtnText; }
        }
    }, 300);
}

// Window 객체에 전역 함수 노출
window.handleChangePassword = handleChangePassword;
window.renderDeviceCard = renderDeviceCard;
window.deactivateDevice = deactivateDevice;
window.reactivateDevice = reactivateDevice;
// window.deleteDevice = deleteDevice;  // 제거됨, cleanup 사용
window.toggleActivityLogs = toggleActivityLogs;
window.copyToClipboard = copyToClipboard;
window.openCleanupModal = openCleanupModal;
window.handleCleanupDevices = handleCleanupDevices;
// window.handleLogin = handleLogin;  // main.js에서 처리됨
// window.handleLogout = handleLogout;  // main.js에서 처리됨

