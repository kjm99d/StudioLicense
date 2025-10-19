import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { formatDateTime, escapeHtml } from '../utils.js';

export async function loadAdmins() {
  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/admins`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const body = await res.json();
    const tbody = document.getElementById('admins-tbody');
    if (!tbody) {
      console.error('admins-tbody element not found');
      return;
    }
    
    if (res.ok && body.status === 'success') {
      const admins = body.data || [];
      console.log('Loaded admins:', admins);
      
      if (admins.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center">관리자가 없습니다.</td></tr>';
      } else {
        const html = admins.map(a => `
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
            <td>
              ${a.role === 'super_admin' ? '-' : `
                <button class="btn btn-sm btn-warning" data-action="reset" data-admin-id="${escapeHtml(a.id)}" data-admin-name="${escapeHtml(a.username)}">🔑 비밀번호 초기화</button>
                <button class="btn btn-sm btn-danger" data-action="delete" data-admin-id="${escapeHtml(a.id)}" data-admin-name="${escapeHtml(a.username)}">🗑️ 삭제</button>
              `}
            </td>
          </tr>
        `).join('');
        
        tbody.innerHTML = html;
        
        // 이벤트 리스너 추가
        tbody.querySelectorAll('button[data-action]').forEach(btn => {
          btn.addEventListener('click', async (e) => {
            e.preventDefault();
            const action = btn.dataset.action;
            const adminId = btn.dataset.adminId;
            const adminName = btn.dataset.adminName;
            
            if (action === 'reset') {
              await resetAdminPassword(adminId, adminName, btn);
            } else if (action === 'delete') {
              await deleteAdmin(adminId, adminName, btn);
            }
          });
        });
        
        console.log('Admin table updated successfully');
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
      // 실패 시 버튼 상태 복구 후 alert
      if (submitBtn) { 
        submitBtn.disabled = originalBtnDisabled; 
        submitBtn.textContent = originalBtnText; 
      }
      await showAlert(body.message || '생성에 실패했습니다.', '관리자 생성 실패');
      return; // 여기서 반환해서 finally에서 중복 복구 방지
    }
  } catch (err) {
    console.error('Failed to create admin:', err);
    // 에러 시 버튼 상태 복구 후 alert
    if (submitBtn) { 
      submitBtn.disabled = originalBtnDisabled; 
      submitBtn.textContent = originalBtnText; 
    }
    await showAlert('서버 오류가 발생했습니다.', '관리자 생성 실패');
    return; // 여기서 반환해서 finally에서 중복 복구 방지
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
