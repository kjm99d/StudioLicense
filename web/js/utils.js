export function formatDate(dateString) {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleDateString('ko-KR', { year: 'numeric', month: '2-digit', day: '2-digit' });
}

export function formatDateTime(dateString) {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleString('ko-KR', {
    year: '2-digit', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  });
}

export function debounce(fn, wait) {
  let t;
  return function(...args) {
    clearTimeout(t);
    t = setTimeout(() => fn.apply(this, args), wait);
  };
}

export function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function formatAdminAction(action) {
  const actionMap = {
    'login': '로그인',
    'change_password': '비밀번호 변경',
    'create_product': '제품 생성',
    'update_product': '제품 수정',
    'delete_product': '제품 삭제',
    'create_license': '라이선스 생성',
    'update_license': '라이선스 수정',
    'delete_license': '라이선스 삭제',
    'create_policy': '정책 생성',
    'update_policy': '정책 수정',
    'delete_policy': '정책 삭제',
    'deactivate_device': '디바이스 비활성화',
    'reactivate_device': '디바이스 활성화',
    'cleanup_devices': '디바이스 정리',
    'create_admin': '관리자 생성',
    'reset_password': '비밀번호 초기화',
    'delete_admin': '관리자 삭제'
  };
  return actionMap[action] || action;
}

export function safeParseJSON(str) {
  try {
    if (!str) return {};
    return JSON.parse(str);
  } catch (_) {
    return {};
  }
}

export function getValidationWarning(lastValidated) {
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

export function renderDeviceStatusBadge(status) {
  const map = {
    'active': '<span class="new badge green" data-badge-caption="">활성</span>',
    'deactivated': '<span class="new badge red" data-badge-caption="">비활성</span>'
  };
  return map[status] || `<span class="new badge grey" data-badge-caption="">${escapeHtml(status || '-')}</span>`;
}

export async function copyToClipboard(elementId) {
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
    const { showAlert } = await import('./modals.js');
    await showAlert('클립보드 복사에 실패했습니다.', '복사 실패');
  }
}
