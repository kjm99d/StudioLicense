import { apiFetch, API_BASE_URL } from '../api.js';
import { closeModal, showAlert } from '../modals.js';

export async function handleChangePassword(e) {
  e.preventDefault();
  const oldPassword = document.getElementById('current_password').value;
  const newPassword = document.getElementById('new_password').value;
  const newPasswordConfirm = document.getElementById('new_password_confirm').value;

  if (newPassword !== newPasswordConfirm) {
    await showAlert('새 비밀번호 확인이 일치하지 않습니다.', '비밀번호 변경');
    return;
  }
  if (!newPassword || newPassword.length < 8) {
    await showAlert('새 비밀번호는 8자 이상이어야 합니다.', '비밀번호 변경');
    return;
  }

  const modal = document.getElementById('change-password-modal');

  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/change-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
      _noGlobalLoading: true
    });

    const result = await response.json();

    // 먼저 모달 닫고 폼 리셋
    if (modal) closeModal(modal);
    e.target.reset();

    if (response.ok && result.status === 'success') {
      setTimeout(async () => {
        await showAlert('비밀번호가 변경되었습니다. 다시 로그인 해주세요.', '비밀번호 변경 완료');
        // 로그아웃 처리 (메인에 노출된 핸들러 사용)
        if (window.handleLogout) window.handleLogout();
      }, 300);
    } else {
      setTimeout(() => {
        showAlert(result.message || '비밀번호 변경에 실패했습니다.', '비밀번호 변경 실패');
      }, 300);
    }
  } catch (error) {
    console.error('Failed to change password:', error);
    if (modal) closeModal(modal);
    e.target.reset();
    setTimeout(() => {
      showAlert('서버 오류가 발생했습니다.', '비밀번호 변경 실패');
    }, 300);
  }
}

// 전역 노출 (현재 HTML 이벤트 연결과 호환)
window.handleChangePassword = handleChangePassword;
