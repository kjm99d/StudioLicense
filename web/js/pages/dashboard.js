import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { formatDateTime, escapeHtml, formatAdminAction } from '../utils.js';

export async function loadDashboardStats() {
  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/dashboard/stats`, { headers: { 'Authorization': `Bearer ${state.token}` } });
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

export async function loadRecentActivities() {
  try {
    const type = document.getElementById('activities-type')?.value || '';
    const action = document.getElementById('activities-action')?.value.trim() || '';
    const limit = document.getElementById('activities-limit')?.value || '20';
    const params = new URLSearchParams();
    if (type) params.set('type', type);
    if (action) params.set('action', action);
    if (limit) params.set('limit', limit);
    const qs = params.toString();
    const response = await apiFetch(`${API_BASE_URL}/api/admin/dashboard/activities${qs ? `?${qs}` : ''}`, { headers: { 'Authorization': `Bearer ${state.token}` } });
    const data = await response.json();
    if (data.status === 'success') {
      const activities = data.data;
      const container = document.getElementById('recent-activities');
      const actionIcons = {
        'activated': '🟢 활성화', 'validated': '✅ 검증', 'deactivated': '🔴 비활성화', 'reactivated': '🔄 재활성화',
        'admin:login': '👤 로그인', 'admin:change_password': '🔐 비밀번호 변경', 'admin:create_admin': '👥 관리자 생성',
        'admin:create_product': '📦 제품 생성', 'admin:update_product': '🛠️ 제품 수정', 'admin:delete_product': '🗑️ 제품 삭제',
        'admin:create_license': '🎫 라이선스 생성', 'admin:update_license': '🛠️ 라이선스 수정',
        'admin:deactivate_device': '👮 디바이스 비활성화', 'admin:reactivate_device': '👮 디바이스 재활성화',
        'admin:cleanup_devices': '🧹 디바이스 정리'
      };
      if (activities && activities.length > 0) {
        container.innerHTML = activities.map(a => {
          if (a.type === 'admin') {
            // 한글로 변환된 action 사용
            const actionLabel = formatAdminAction(a.action);
            const iconMap = {
              'login': '👤',
              'change_password': '🔐',
              'create_admin': '👥',
              'reset_password': '🔑',
              'delete_admin': '🗑️',
              'create_product': '📦',
              'update_product': '🛠️',
              'delete_product': '🗑️',
              'create_license': '🎫',
              'update_license': '🛠️',
              'delete_license': '🗑️',
              'create_policy': '🛡️',
              'update_policy': '✏️',
              'delete_policy': '🗑️',
              'deactivate_device': '👮',
              'reactivate_device': '👮',
              'cleanup_devices': '🧹',
              'upload_file': '📤',
              'delete_file': '🗑️',
              'download_file': '📥',
              'attach_product_file': '📝',
              'update_product_file': '🛠️',
              'delete_product_file': '🗑️',
              'delete_device': '🗑️'
            };
            const icon = iconMap[a.action] || '👤';
            const label = `${icon} ${actionLabel}`;
            const details = a.details ? `<div style="color:#6b7280; font-size: 0.9em; margin-top: 4px;">${escapeHtml(a.details)}</div>` : '';
            return `<div class="activity-item"><div><div><strong>관리자</strong> · ${escapeHtml(a.admin_username || '-')}</div><div style="margin-top:2px; color:#374151;">${label}</div>${details}</div><div><small>${formatDateTime(a.created_at)}</small></div></div>`;
          } else {
            const actionLabel = actionIcons[a.action] || `📝 ${a.action}`;
            const product = a.product_name ? ` <span class="badge">${escapeHtml(a.product_name)}</span>` : '';
            const fp = a.fingerprint ? `<code style="font-size: 0.8em;">${escapeHtml(a.fingerprint)}</code>` : '';
            const details = a.details ? `<div style="color:#6b7280; font-size: 0.9em; margin-top: 4px;">${escapeHtml(a.details)}</div>` : '';
            return `<div class="activity-item"><div><div><strong>${escapeHtml(a.customer_name || '-')}</strong> - ${escapeHtml(a.license_key || '-')}${product}</div><div style="margin-top:2px; color:#374151;">${actionLabel} · ${escapeHtml(a.device_name || '-')} ${fp}</div>${details}</div><div><small>${formatDateTime(a.created_at)}</small></div></div>`;
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
