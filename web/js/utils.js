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
