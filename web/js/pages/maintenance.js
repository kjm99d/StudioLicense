import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';

export function openCleanupModal() {
  const modal = document.getElementById('cleanup-modal');
  const daysInput = document.getElementById('cleanup-days');
  if (daysInput) daysInput.value = '90';
  const preview = document.getElementById('cleanup-preview');
  if (preview) preview.style.display = 'none';
  if (modal) openModal(modal);
}

export async function handleCleanupDevices(e) {
  const days = parseInt(document.getElementById('cleanup-days').value);

  if (isNaN(days) || days < 0) {
    await showAlert('0 이상의 유효한 일수를 입력해주세요.', '입력 오류');
    return;
  }

  const message = days === 0
    ? '모든 비활성 디바이스를 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.'
    : `비활성화된 지 ${days}일이 지난 디바이스를 모두 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.`;

  const cleanupModal = document.getElementById('cleanup-modal');
  if (cleanupModal) closeModal(cleanupModal);

  setTimeout(async () => {
    const ok = await showConfirm(message, '디바이스 정리');
    if (!ok) {
      if (cleanupModal) openModal(cleanupModal);
      return;
    }

    const submitBtn = e && e.target ? e.target : document.getElementById('cleanup-confirm-btn');
    const originalBtnText = submitBtn ? submitBtn.textContent : '';
    if (submitBtn) { submitBtn.disabled = true; submitBtn.textContent = '정리 중...'; }

    try {
      const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/cleanup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ days }),
        _noGlobalLoading: true
      });
      const result = await response.json();

      if (response.ok && result.status === 'success') {
        const count = result.data ? result.data.deleted_count : 0;
        showAlert(`${count}개의 디바이스가 삭제되었습니다.`, '디바이스 정리 완료');
        // 대시보드가 열려있다면 업데이트
        const dash = document.getElementById('dashboard-content');
        if (dash && dash.classList.contains('active')) {
          if (window.loadDashboardStats) window.loadDashboardStats();
          if (window.loadRecentActivities) window.loadRecentActivities();
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

// 전역 노출 (현재 HTML/메인 바인딩 호환)
window.openCleanupModal = openCleanupModal;
window.handleCleanupDevices = handleCleanupDevices;
