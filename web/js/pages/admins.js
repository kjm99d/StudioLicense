import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { formatDateTime, escapeHtml } from '../utils.js';

export async function loadAdmins() {
  try {
    const tbody = document.getElementById('admins-tbody');
    if (!tbody) {
      console.error('admins-tbody element not found');
      return;
    }
    // 로딩 상태 표시 (요청 시작 전)
    tbody.innerHTML = '<tr><td colspan="5" class="text-center">로딩 중...</td></tr>';

    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const body = await res.json();
    
    if (res.ok && body.status === 'success') {
      const admins = body.data || [];
      console.log('Loaded admins:', admins);
      
      // 역할 정규화 헬퍼
      const isSuper = (role) => {
        if (!role) return false;
        return String(role).toLowerCase().replace(/-/g, '_') === 'super_admin';
      };
      
      if (admins.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center">관리자가 없습니다.</td></tr>';
      } else {
        // DOM API로 안전하게 렌더링하여 셀 누락 문제를 방지
        tbody.innerHTML = '';
        admins.forEach(a => {
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
      tbody.innerHTML = `<tr><td colspan="5" class="text-center">불러오기에 실패했습니다: ${escapeHtml(body.message || '')}</td></tr>`;
    }
  } catch (e) {
    console.error('Failed to load admins:', e);
    const tbody = document.getElementById('admins-tbody');
    if (tbody) tbody.innerHTML = '<tr><td colspan="5" class="text-center">서버 오류</td></tr>';
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
      // 순서 중요: 먼저 데이터 로드, 그 다음 UI 업데이트
      await loadAdmins();
      if (window.loadRecentActivities) await window.loadRecentActivities();
      
      // 모달 닫기 및 폼 초기화 (alert 전에)
      const createAdminModal = document.getElementById('create-admin-modal');
      if (createAdminModal) {
        closeModal(createAdminModal);
      }
      e.target.reset();
      
      // 모달 닫은 후 alert 보이기
      setTimeout(() => {
        showAlert('서브 관리자가 생성되었습니다.', '관리자 생성');
      }, 300);
    } else {
      // 실패 시: 먼저 모달 닫기, 그 다음 alert 표시
      const createAdminModal = document.getElementById('create-admin-modal');
      if (createAdminModal) {
        closeModal(createAdminModal);
      }
      e.target.reset();
      
      // 버튼 상태 복구
      if (submitBtn) { 
        submitBtn.disabled = originalBtnDisabled; 
        submitBtn.textContent = originalBtnText; 
      }
      
      // 모달 닫은 후 alert 보이기
      setTimeout(() => {
        showAlert(body.message || '생성에 실패했습니다.', '관리자 생성 실패');
      }, 300);
      return;
    }
  } catch (err) {
    console.error('Failed to create admin:', err);
    
    // 에러 시: 먼저 모달 닫기, 그 다음 alert 표시
    const createAdminModal = document.getElementById('create-admin-modal');
    if (createAdminModal) {
      closeModal(createAdminModal);
    }
    e.target.reset();
    
    // 버튼 상태 복구
    if (submitBtn) { 
      submitBtn.disabled = originalBtnDisabled; 
      submitBtn.textContent = originalBtnText; 
    }
    
    // 모달 닫은 후 alert 보이기
    setTimeout(() => {
      showAlert('서버 오류가 발생했습니다.', '관리자 생성 실패');
    }, 300);
    return;
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
window.resetAdminPassword = resetAdminPassword;
window.deleteAdmin = deleteAdmin;
