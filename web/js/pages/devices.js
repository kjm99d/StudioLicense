import { apiFetch, API_BASE_URL } from '../api.js';
import { showAlert, showConfirm } from '../modals.js';
import { formatDate, escapeHtml, safeParseJSON, getValidationWarning, renderDeviceStatusBadge, copyToClipboard } from '../utils.js';
import { hasPermission } from '../state.js';
import { PERMISSIONS } from '../permissions.js';

// 디바이스 카드 렌더링
export function renderDeviceCard(d) {
  const info = safeParseJSON(d.device_info);
  const statusBadge = renderDeviceStatusBadge(d.status);
  const validationWarning = getValidationWarning(d.last_validated_at);
  const isActive = d.status === 'active';
  const licenseId = d.license_id;
  const canManageDevices = hasPermission(PERMISSIONS.DEVICES_MANAGE);
  const canViewDevices = hasPermission(PERMISSIONS.DEVICES_VIEW) || canManageDevices;
  const manageButton = canManageDevices
    ? (isActive
        ? `<button class="btn btn-sm btn-danger" onclick="deactivateDevice('${d.id}', '${escapeHtml(d.device_name)}', '${licenseId}')">비활성화</button>`
        : `<button class="btn btn-sm btn-success" onclick="reactivateDevice('${d.id}', '${escapeHtml(d.device_name)}', '${licenseId}')">재활성화</button>`
      )
    : '';
  const logsButton = canViewDevices
    ? `<button class="btn btn-sm" onclick="toggleActivityLogs('${d.id}')">📋 활동 로그</button>`
    : '';
  const logsSection = canViewDevices ? `<div style="margin-top: 8px;">${logsButton}</div>` : '';

  return `
  <div class="device-card ${isActive ? '' : 'inactive'} card">
    <div class="device-card-header">
      <div class="device-name">💻 <strong>${escapeHtml(d.device_name || '이름 없음')}</strong></div>
      <div class="device-actions">
        ${statusBadge}
        ${manageButton}
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
      ${logsSection}
      <div id="activity-logs-${d.id}" class="activity-logs" style="display: none;"></div>
    </div>
  </div>`;
}

// 디바이스 비활성화
export async function deactivateDevice(deviceId, deviceName, licenseId) {
  if (!hasPermission(PERMISSIONS.DEVICES_MANAGE)) {
    await showAlert('디바이스를 관리할 권한이 없습니다.', '권한 부족');
    return;
  }

  const ok = await showConfirm(
    `"${deviceName}" 디바이스를 비활성화하시겠습니까?\n\n비활성화하면 이 디바이스에서 더 이상 라이선스를 사용할 수 없습니다.`,
    '디바이스 비활성화'
  );
  
  if (!ok) return;
  
  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/deactivate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceId }),
      _noGlobalLoading: true
    });
    
    const result = await response.json();
    
    if (response.ok && result.status === 'success') {
      await showAlert('디바이스가 비활성화되었습니다.\n\n해당 디바이스에서는 더 이상 라이선스 검증이 실패하며, 최대 디바이스 슬롯 하나가 해제되어 새로운 디바이스를 등록할 수 있습니다.', '디바이스 비활성화');
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
export async function reactivateDevice(deviceId, deviceName, licenseId) {
  if (!hasPermission(PERMISSIONS.DEVICES_MANAGE)) {
    await showAlert('디바이스를 관리할 권한이 없습니다.', '권한 부족');
    return;
  }

  const ok = await showConfirm(
    `"${deviceName}" 디바이스를 재활성화하시겠습니까?\n\n재활성화하면 이 디바이스에서 다시 라이선스를 사용할 수 있습니다. (디바이스 슬롯이 남아있어야 합니다)`,
    '디바이스 재활성화'
  );
  
  if (!ok) return;
  
  try {
    const response = await apiFetch(`${API_BASE_URL}/api/admin/devices/reactivate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceId }),
      _noGlobalLoading: true
    });
    
    const result = await response.json();
    
    if (response.ok && result.status === 'success') {
      await showAlert('디바이스가 재활성화되었습니다.\n\n해당 디바이스에서 라이선스 검증이 다시 정상적으로 작동합니다.', '디바이스 재활성화');
      await reloadDeviceList(licenseId);
    } else {
      await showAlert(result.message || '디바이스 재활성화에 실패했습니다.', '디바이스 재활성화');
    }
  } catch (error) {
    console.error('Failed to reactivate device:', error);
    await showAlert('서버 오류가 발생했습니다.', '디바이스 재활성화');
  }
}

// 활동 로그 토글
export async function toggleActivityLogs(deviceId) {
  if (!hasPermission(PERMISSIONS.DEVICES_VIEW) && !hasPermission(PERMISSIONS.DEVICES_MANAGE)) {
    await showAlert('디바이스 로그를 볼 권한이 없습니다.', '권한 부족');
    return;
  }

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
      _noGlobalLoading: true
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
        <span style="color: #999; font-size: 0.9em;">${formatDate(log.created_at)}</span>
      </div>
      ${details}
    </div>`;
  }).join('');
  
  return `<div class="activity-log-list">${items}</div>`;
}

// 디바이스 목록 부분 리로드
export async function reloadDeviceList(licenseId) {
  try {
    // 디바이스 목록 갱신
    const res = await apiFetch(`${API_BASE_URL}/api/admin/licenses/devices?id=${licenseId}`, {
      _noGlobalLoading: true
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
    const licenseRes = await apiFetch(`${API_BASE_URL}/api/admin/licenses/${encodeURIComponent(licenseId)}`, {
      _noGlobalLoading: true
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
      window.loadLicenses(window.state?.currentPage || 1);
    }
  } catch (e) {
    console.error('Failed to reload device list:', e);
  }
}

// 전역 노출 (HTML onclick 호환)
window.renderDeviceCard = renderDeviceCard;
window.deactivateDevice = deactivateDevice;
window.reactivateDevice = reactivateDevice;
window.toggleActivityLogs = toggleActivityLogs;
window.copyToClipboard = copyToClipboard;
