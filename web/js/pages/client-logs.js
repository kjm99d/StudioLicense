// 클라이언트 로그 관리 페이지
import { getAPI, deleteAPI } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { state } from '../state.js';

let currentPage = 1;
const pageSize = 50;
let currentFilters = {};

// 클라이언트 로그 목록 로드
export async function loadClientLogs(page = 1) {
    try {
        const params = new URLSearchParams({
            page: page,
            page_size: pageSize,
            ...currentFilters
        });

        const response = await getAPI(`/api/admin/client-logs?${params.toString()}`);
        const data = await response.json();
        
        if (data.status === 'success') {
            displayClientLogs(data.data, data.meta);
            currentPage = page;
        }
    } catch (error) {
        console.error('로그 로드 실패:', error);
        document.getElementById('client-logs-tbody').innerHTML = 
            '<tr><td colspan="7" class="error">로그를 불러오는데 실패했습니다.</td></tr>';
    }
}

// 클라이언트 로그 표시
function displayClientLogs(logs, meta) {
    const tbody = document.getElementById('client-logs-tbody');
    
    if (!logs || logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="empty">로그가 없습니다.</td></tr>';
        document.getElementById('logs-pagination').innerHTML = '';
        return;
    }

    tbody.innerHTML = logs.map(log => `
        <tr onclick="showLogDetail(${log.id})" style="cursor: pointer;">
            <td>${formatDateTime(log.created_at)}</td>
            <td><span class="badge ${getLogLevelClass(log.level)}">${log.level}</span></td>
            <td><span class="badge" style="background-color: #9e9e9e;">${log.category}</span></td>
            <td>${truncate(log.license_key, 20)}</td>
            <td>${log.device_id || '-'}</td>
            <td>${escapeHtml(truncate(log.message, 50))}</td>
            <td>
                <button class="btn-small blue" onclick="event.stopPropagation(); showLogDetail(${log.id})">
                    <i class="material-icons">visibility</i>
                </button>
            </td>
        </tr>
    `).join('');

    // 페이지네이션 렌더링
    renderLogsPagination(meta);
}

// 로그 레벨에 따른 CSS 클래스
function getLogLevelClass(level) {
    switch (level) {
        case 'DEBUG': return 'grey';
        case 'INFO': return 'blue';
        case 'WARN': return 'orange';
        case 'ERROR': return 'red';
        case 'FATAL': return 'red darken-3';
        default: return 'grey';
    }
}

// 로그 상세 보기
window.showLogDetail = async function(logId) {
    try {
        const params = new URLSearchParams(currentFilters);
        const response = await getAPI(`/api/admin/client-logs?${params.toString()}`);
        const data = await response.json();
        
        if (data.status === 'success') {
            const log = data.data.find(l => l.id === logId);
            
            if (log) {
                const content = document.getElementById('log-detail-content');
                content.innerHTML = `
                    <div style="display: grid; gap: 15px;">
                        <div>
                            <strong style="color: #666;">시간:</strong>
                            <div style="margin-top: 5px;">${formatDateTime(log.created_at)}</div>
                        </div>
                        <div>
                            <strong style="color: #666;">레벨:</strong>
                            <div style="margin-top: 5px;">
                                <span class="badge ${getLogLevelClass(log.level)}">${log.level}</span>
                            </div>
                        </div>
                        <div>
                            <strong style="color: #666;">카테고리:</strong>
                            <div style="margin-top: 5px;">
                                <span class="badge" style="background-color: #9e9e9e;">${log.category}</span>
                            </div>
                        </div>
                        <div>
                            <strong style="color: #666;">라이선스 키:</strong>
                            <div style="margin-top: 5px; font-family: monospace;">${escapeHtml(log.license_key)}</div>
                        </div>
                        ${log.device_id ? `
                        <div>
                            <strong style="color: #666;">디바이스 ID:</strong>
                            <div style="margin-top: 5px; font-family: monospace;">${escapeHtml(log.device_id)}</div>
                        </div>
                        ` : ''}
                        <div>
                            <strong style="color: #666;">메시지:</strong>
                            <div style="margin-top: 5px; padding: 10px; background: #f5f5f5; border-radius: 4px;">
                                ${escapeHtml(log.message)}
                            </div>
                        </div>
                        ${log.details ? `
                        <div>
                            <strong style="color: #666;">상세 정보:</strong>
                            <div style="margin-top: 5px; padding: 10px; background: #f5f5f5; border-radius: 4px; white-space: pre-wrap; font-family: monospace; font-size: 12px;">
                                ${escapeHtml(log.details)}
                            </div>
                        </div>
                        ` : ''}
                        ${log.stack_trace ? `
                        <div>
                            <strong style="color: #666;">스택 트레이스:</strong>
                            <div style="margin-top: 5px; padding: 10px; background: #fff3cd; border-radius: 4px; white-space: pre-wrap; font-family: monospace; font-size: 11px; max-height: 200px; overflow-y: auto;">
                                ${escapeHtml(log.stack_trace)}
                            </div>
                        </div>
                        ` : ''}
                        ${log.app_version ? `
                        <div>
                            <strong style="color: #666;">앱 버전:</strong>
                            <div style="margin-top: 5px;">${escapeHtml(log.app_version)}</div>
                        </div>
                        ` : ''}
                        ${log.os_version ? `
                        <div>
                            <strong style="color: #666;">OS 버전:</strong>
                            <div style="margin-top: 5px;">${escapeHtml(log.os_version)}</div>
                        </div>
                        ` : ''}
                        ${log.client_ip ? `
                        <div>
                            <strong style="color: #666;">클라이언트 IP:</strong>
                            <div style="margin-top: 5px; font-family: monospace;">${escapeHtml(log.client_ip)}</div>
                        </div>
                        ` : ''}
                    </div>
                `;
                
                openModal(document.getElementById('log-detail-modal'));
            }
        }
    } catch (error) {
        console.error('로그 상세 조회 실패:', error);
        showAlert('로그 상세 정보를 불러오는데 실패했습니다.');
    }
};

// 필터 적용
window.applyClientLogsFilter = function() {
    currentFilters = {};
    
    const licenseKey = document.getElementById('filter-license-key').value.trim();
    const deviceId = document.getElementById('filter-device-id').value.trim();
    const level = document.getElementById('filter-level').value;
    const category = document.getElementById('filter-category').value;
    
    if (licenseKey) currentFilters.license_key = licenseKey;
    if (deviceId) currentFilters.device_id = deviceId;
    if (level) currentFilters.level = level;
    if (category) currentFilters.category = category;
    
    loadClientLogs(1);
};

// 로그 정리 모달 열기
window.openCleanupLogsModal = function() {
    const form = document.getElementById('cleanup-logs-form');
    form.reset();
    
    // 기본값: 30일 전
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
    document.getElementById('cleanup_before_date').value = thirtyDaysAgo.toISOString().split('T')[0];
    
    openModal(document.getElementById('cleanup-logs-modal'));
};

// 로그 정리 폼 제출
document.getElementById('cleanup-logs-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const beforeDate = document.getElementById('cleanup_before_date').value;
    
    // 먼저 cleanup 모달 닫기 (confirm 모달과 충돌 방지)
    const cleanupModal = document.getElementById('cleanup-logs-modal');
    if (cleanupModal) {
        closeModal(cleanupModal);
    }
    
    // 모달 닫힌 후 confirm 표시
    setTimeout(async () => {
        const confirmed = await showConfirm(
            `${beforeDate} 이전의 모든 로그를 삭제하시겠습니까?\n이 작업은 되돌릴 수 없습니다.`,
            '로그 삭제 확인'
        );
        
        if (!confirmed) {
            // 취소 시 모달 다시 열기
            if (cleanupModal) {
                openModal(cleanupModal);
            }
            return;
        }
        
        try {
            const response = await deleteAPI(`/api/admin/client-logs/cleanup?before_date=${beforeDate}`);
            const data = await response.json();
            
            if (data.status === 'success') {
                showAlert(`${data.data.deleted_count}개의 로그가 삭제되었습니다.`, '로그 삭제 완료');
                loadClientLogs(1);
            } else {
                showAlert(data.message || '로그 삭제에 실패했습니다.', '로그 삭제 실패');
            }
        } catch (error) {
            console.error('로그 삭제 실패:', error);
            showAlert('로그 삭제에 실패했습니다.', '로그 삭제 실패');
        }
    }, 300);
});

// 페이지네이션 렌더링
function renderLogsPagination(meta) {
    const container = document.getElementById('logs-pagination');
    
    if (!meta || meta.total_pages <= 1) {
        container.innerHTML = '';
        return;
    }

    const { page, total_pages } = meta;
    let html = '<ul class="pagination">';

    // 이전 버튼
    if (page > 1) {
        html += `<li class="waves-effect"><a href="#" onclick="loadClientLogs(${page - 1})"><i class="material-icons">chevron_left</i></a></li>`;
    } else {
        html += `<li class="disabled"><a href="#"><i class="material-icons">chevron_left</i></a></li>`;
    }

    // 페이지 번호
    const maxVisible = 7;
    let startPage = Math.max(1, page - Math.floor(maxVisible / 2));
    let endPage = Math.min(total_pages, startPage + maxVisible - 1);

    if (endPage - startPage < maxVisible - 1) {
        startPage = Math.max(1, endPage - maxVisible + 1);
    }

    if (startPage > 1) {
        html += `<li class="waves-effect"><a href="#" onclick="loadClientLogs(1)">1</a></li>`;
        if (startPage > 2) {
            html += `<li class="disabled"><a href="#">...</a></li>`;
        }
    }

    for (let i = startPage; i <= endPage; i++) {
        if (i === page) {
            html += `<li class="active"><a href="#">${i}</a></li>`;
        } else {
            html += `<li class="waves-effect"><a href="#" onclick="loadClientLogs(${i})">${i}</a></li>`;
        }
    }

    if (endPage < total_pages) {
        if (endPage < total_pages - 1) {
            html += `<li class="disabled"><a href="#">...</a></li>`;
        }
        html += `<li class="waves-effect"><a href="#" onclick="loadClientLogs(${total_pages})">${total_pages}</a></li>`;
    }

    // 다음 버튼
    if (page < total_pages) {
        html += `<li class="waves-effect"><a href="#" onclick="loadClientLogs(${page + 1})"><i class="material-icons">chevron_right</i></a></li>`;
    } else {
        html += `<li class="disabled"><a href="#"><i class="material-icons">chevron_right</i></a></li>`;
    }

    html += '</ul>';
    container.innerHTML = html;
}

// 유틸리티 함수
function formatDateTime(dateStr) {
    if (!dateStr) return '-';
    return dateStr.replace('T', ' ').substring(0, 19);
}

function truncate(str, maxLen) {
    if (!str) return '-';
    return str.length > maxLen ? str.substring(0, maxLen) + '...' : str;
}

function escapeHtml(text) {
    if (!text) return '';
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, m => map[m]);
}

// 페이지 초기화
export function initClientLogsPage() {
    loadClientLogs(1);
}

// 전역 함수 노출
window.loadClientLogs = loadClientLogs;
