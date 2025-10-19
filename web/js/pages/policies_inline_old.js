// 정책 관리 페이지
import { getAPI, postAPI, putAPI, deleteAPI } from '../api.js';
import { formatDate } from '../utils.js';

let policies = [];

// 정책 페이지 초기화
export async function initPoliciesPage() {
    console.log('🔧 Initializing policies page...');
    const container = document.getElementById('page-content');
    
    if (!container) {
        console.error('❌ page-content element not found!');
        return;
    }
    
    container.innerHTML = `
        <style>
        /* 전체 컨테이너 */
        .policies-container {
            padding: 30px;
            max-width: 1400px;
            margin: 0 auto;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
        }
        
        /* 헤더 영역 */
        .policies-container .page-header {
            margin-bottom: 32px;
        }
        .policies-container .page-header h2 {
            margin: 0 0 8px 0;
            color: #1a1a1a;
            font-size: 32px;
            font-weight: 700;
        }
        .policies-container .subtitle {
            color: #6c757d;
            font-size: 15px;
            margin: 0;
        }
        
        /* 그리드 레이아웃 */
        .policies-content {
            display: grid;
            grid-template-columns: 420px 1fr;
            gap: 24px;
        }
        @media (max-width: 1200px) {
            .policies-content {
                grid-template-columns: 1fr;
            }
        }
        
        /* 정책 생성 폼 섹션 */
        .policy-form-section {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 28px;
            border-radius: 16px;
            box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
            height: fit-content;
            position: sticky;
            top: 20px;
        }
        .policy-form-section h3 {
            margin: 0 0 24px 0;
            font-size: 20px;
            color: white;
            font-weight: 600;
        }
        
        /* 폼 스타일 */
        .policy-form {
            display: flex;
            flex-direction: column;
        }
        .form-group {
            margin-bottom: 20px;
        }
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            font-size: 13px;
            color: rgba(255,255,255,0.95);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .form-group input,
        .form-group textarea,
        .form-group select {
            width: 100%;
            padding: 12px 16px;
            border: 2px solid rgba(255,255,255,0.2);
            border-radius: 10px;
            font-size: 14px;
            box-sizing: border-box;
            font-family: inherit;
            transition: all 0.3s ease;
            background: rgba(255,255,255,0.95);
            color: #333;
        }
        .form-group input:focus,
        .form-group textarea:focus,
        .form-group select:focus {
            outline: none;
            border-color: rgba(255,255,255,0.8);
            box-shadow: 0 0 0 4px rgba(255,255,255,0.2);
            background: white;
        }
        .form-group textarea {
            font-family: 'Courier New', monospace;
            resize: vertical;
            min-height: 120px;
        }
        .form-group input::placeholder,
        .form-group textarea::placeholder {
            color: #999;
        }
        
        /* 버튼 스타일 */
        .btn {
            padding: 12px 24px;
            border: none;
            border-radius: 10px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            display: inline-flex;
            align-items: center;
            justify-content: center;
            letter-spacing: 0.3px;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 20px rgba(0,0,0,0.2);
        }
        .btn:active {
            transform: translateY(0);
        }
        .btn-primary {
            background: white;
            color: #667eea;
            font-weight: 700;
        }
        .btn-primary:hover {
            background: #f8f9ff;
            box-shadow: 0 8px 25px rgba(255,255,255,0.4);
        }
        .btn-secondary {
            background: rgba(255,255,255,0.2);
            color: white;
            border: 2px solid rgba(255,255,255,0.3);
        }
        .btn-secondary:hover {
            background: rgba(255,255,255,0.3);
            border-color: rgba(255,255,255,0.5);
        }
        .btn-danger {
            background: linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%);
            color: white;
        }
        .btn-danger:hover {
            background: linear-gradient(135deg, #ee5a6f 0%, #d63447 100%);
        }
        .btn-sm {
            padding: 8px 16px;
            font-size: 13px;
        }
        
        /* 정책 목록 섹션 */
        .policy-list-section h3 {
            margin: 0 0 20px 0;
            font-size: 20px;
            color: #1a1a1a;
            font-weight: 700;
        }
        .policies-list {
            display: flex;
            flex-direction: column;
            gap: 16px;
        }
        
        /* 정책 아이템 카드 */
        .policy-item {
            background: white;
            border: 2px solid #f0f0f0;
            border-radius: 16px;
            padding: 24px;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            position: relative;
            overflow: hidden;
        }
        .policy-item::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            transform: scaleX(0);
            transition: transform 0.3s ease;
        }
        .policy-item:hover {
            transform: translateY(-4px);
            box-shadow: 0 12px 40px rgba(0,0,0,0.12);
            border-color: #667eea;
        }
        .policy-item:hover::before {
            transform: scaleX(1);
        }
        
        /* 정책 정보 */
        .policy-info h4 {
            margin: 0 0 8px 0;
            font-size: 20px;
            color: #1a1a1a;
            font-weight: 700;
        }
        .policy-id {
            font-size: 12px;
            color: #999;
            margin: 0 0 12px 0;
            font-family: 'Courier New', monospace;
            background: #f8f9fa;
            padding: 4px 8px;
            border-radius: 4px;
            display: inline-block;
        }
        .policy-status {
            margin-bottom: 16px;
        }
        
        /* 상태 배지 */
        .status-badge {
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            display: inline-block;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .status-badge.active {
            background: linear-gradient(135deg, #51cf66 0%, #37b24d 100%);
            color: white;
            box-shadow: 0 4px 12px rgba(81, 207, 102, 0.3);
        }
        .status-badge.inactive {
            background: linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%);
            color: white;
            box-shadow: 0 4px 12px rgba(255, 107, 107, 0.3);
        }
        
        /* 정책 데이터 미리보기 */
        .policy-data-preview {
            margin: 16px 0;
            background: #f8f9fa;
            border-radius: 12px;
            border: 1px solid #e9ecef;
            overflow: hidden;
        }
        .policy-data-preview pre {
            background: transparent;
            padding: 16px;
            margin: 0;
            overflow-x: auto;
            font-size: 12px;
            font-family: 'Courier New', Monaco, monospace;
            color: #495057;
            line-height: 1.6;
        }
        
        /* 정책 메타 정보 */
        .policy-meta {
            font-size: 12px;
            color: #868e96;
            margin: 16px 0 0 0;
            padding-top: 16px;
            border-top: 1px solid #f0f0f0;
        }
        
        /* 정책 액션 버튼 */
        .policy-actions {
            margin-top: 20px;
            display: flex;
            gap: 10px;
        }
        
        /* 모달 스타일 */
        .modal {
            display: none;
            position: fixed;
            z-index: 9999;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0,0,0,0.7);
            overflow-y: auto;
            backdrop-filter: blur(4px);
            animation: fadeIn 0.3s ease;
        }
        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }
        .modal.active {
            display: flex !important;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        
        /* 모달 콘텐츠 */
        .modal-content {
            background: white;
            margin: auto;
            padding: 0;
            border-radius: 20px;
            width: 90%;
            max-width: 700px;
            min-height: 500px;
            box-shadow: 0 25px 80px rgba(0,0,0,0.4);
            animation: modalSlideIn 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
            overflow: visible;
            display: flex;
            flex-direction: column;
        }
        @keyframes modalSlideIn {
            from {
                opacity: 0;
                transform: scale(0.9) translateY(-30px);
            }
            to {
                opacity: 1;
                transform: scale(1) translateY(0);
            }
        }
        
        /* 모달 헤더 */
        .modal-header {
            padding: 28px 32px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .modal-header h3 {
            margin: 0;
            font-size: 22px;
            color: white;
            font-weight: 700;
        }
        
        /* 모달 닫기 버튼 */
        .modal-close {
            background: rgba(255,255,255,0.2);
            border: none;
            font-size: 24px;
            cursor: pointer;
            color: white;
            padding: 0;
            width: 36px;
            height: 36px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 8px;
            transition: all 0.2s ease;
        }
        .modal-close:hover {
            background: rgba(255,255,255,0.3);
            transform: rotate(90deg);
        }
        
        /* 모달 바디 */
        .modal-body {
            padding: 40px;
            flex: 1;
            overflow-y: auto;
            min-height: 300px;
        }
        
        /* 모달 폼 스타일 */
        .modal-body .form-group {
            margin-bottom: 24px;
        }
        .modal-body .form-group:last-child {
            margin-bottom: 0;
        }
        .modal-body .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            font-size: 13px;
            color: #495057;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .modal-body .form-group input,
        .modal-body .form-group textarea,
        .modal-body .form-group select {
            width: 100%;
            padding: 12px 16px;
            border: 2px solid #e9ecef;
            border-radius: 10px;
            font-size: 14px;
            box-sizing: border-box;
            font-family: inherit;
            transition: all 0.3s ease;
            background: white;
            color: #333;
        }
        .modal-body .form-group input:focus,
        .modal-body .form-group textarea:focus,
        .modal-body .form-group select:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.1);
        }
        .modal-body .form-group textarea {
            font-family: 'Courier New', monospace;
            resize: vertical;
            min-height: 200px;
            line-height: 1.6;
        }
        .modal-body .form-group select {
            cursor: pointer;
            background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23333' d='M10.293 3.293L6 7.586 1.707 3.293A1 1 0 00.293 4.707l5 5a1 1 0 001.414 0l5-5a1 1 0 10-1.414-1.414z'/%3E%3C/svg%3E");
            background-repeat: no-repeat;
            background-position: right 16px center;
            padding-right: 40px;
            appearance: none;
        }
        
        /* 모달 푸터 */
        .modal-footer {
            padding: 24px 40px;
            background: #f8f9fa;
            display: flex;
            gap: 12px;
            justify-content: flex-end;
            border-top: 1px solid #e9ecef;
            flex-shrink: 0;
        }
        
        /* 빈 상태 */
        .no-data, .loading {
            text-align: center;
            color: #868e96;
            padding: 60px 40px;
            background: white;
            border-radius: 16px;
            border: 2px dashed #dee2e6;
            font-size: 15px;
        }
        </style>
        
        <div class="policies-container">
            <div class="page-header">
                <h2>🛡️ 정책 관리</h2>
                <p class="subtitle">시스템 정책을 생성하고 관리합니다</p>
            </div>

            <div class="policies-content">
                <div class="policy-form-section">
                    <h3>✨ 새 정책 생성</h3>
                    <form id="policyForm" class="policy-form">
                        <div class="form-group">
                            <label for="policyName">정책명</label>
                            <input type="text" id="policyName" placeholder="예: 프리미엄 정책" required />
                        </div>

                        <div class="form-group">
                            <label for="policyData">정책 데이터 (JSON)</label>
                            <textarea id="policyData" placeholder='{"feature": "enabled"}' rows="6" required></textarea>
                        </div>

                        <button type="submit" class="btn btn-primary">🚀 정책 생성</button>
                    </form>
                </div>

                <div class="policy-list-section">
                    <h3>📋 정책 목록</h3>
                    <div id="policiesList" class="policies-list">
                        <p class="loading">정책을 불러오는 중...</p>
                    </div>
                </div>
            </div>
        </div>

        <!-- 정책 수정 모달 -->
        <div id="editPolicyModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h3>✏️ 정책 수정</h3>
                    <button type="button" class="modal-close">&times;</button>
                </div>
                <div class="modal-body">
                    <form id="editPolicyForm">
                        <div class="form-group">
                            <label for="editPolicyName">정책명</label>
                            <input type="text" id="editPolicyName" placeholder="정책명을 입력하세요" />
                        </div>

                        <div class="form-group">
                            <label for="editPolicyData">정책 데이터 (JSON)</label>
                            <textarea id="editPolicyData" rows="10" placeholder='{"feature": "value"}'></textarea>
                        </div>

                        <div class="form-group">
                            <label for="editPolicyStatus">상태</label>
                            <select id="editPolicyStatus">
                                <option value="active">✅ 활성</option>
                                <option value="inactive">❌ 비활성</option>
                            </select>
                        </div>
                    </form>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-action="close">취소</button>
                    <button type="button" class="btn btn-primary" data-action="save">💾 저장</button>
                </div>
            </div>
        </div>
    `;

    setupPolicyEventListeners();
    await loadPolicies();
}

// 정책 목록 로드
async function loadPolicies() {
    try {
        const response = await getAPI('/api/admin/policies');
        const result = await response.json();
        policies = result.data || [];
        renderPoliciesList();
    } catch (error) {
        console.error('Failed to load policies:', error);
        showNotification('정책 조회 실패', 'error');
    }
}

// 정책 목록 렌더링
function renderPoliciesList() {
    const container = document.getElementById('policiesList');
    
    if (policies.length === 0) {
        container.innerHTML = '<p class="no-data">📭 등록된 정책이 없습니다</p>';
        return;
    }

    container.innerHTML = policies.map(policy => `
        <div class="policy-item">
            <div class="policy-info">
                <h4>${escapeHtml(policy.policy_name)}</h4>
                <span class="policy-id">ID: ${policy.id}</span>
                <div class="policy-status">
                    <span class="status-badge ${policy.status}">
                        ${policy.status === 'active' ? '✅ 활성' : '❌ 비활성'}
                    </span>
                </div>
                <div class="policy-data-preview">
                    <pre>${escapeHtml(JSON.stringify(JSON.parse(policy.policy_data), null, 2))}</pre>
                </div>
                <p class="policy-meta">
                    📅 생성: ${formatDate(policy.created_at)} | 🔄 수정: ${formatDate(policy.updated_at)}
                </p>
            </div>
            <div class="policy-actions">
                <button class="btn btn-sm btn-secondary" data-action="edit" data-policy-id="${policy.id}">✏️ 수정</button>
                <button class="btn btn-sm btn-danger" data-action="delete" data-policy-id="${policy.id}">🗑️ 삭제</button>
            </div>
        </div>
    `).join('');

    container.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', handlePolicyAction);
    });
}

// 정책 액션 처리
async function handlePolicyAction(e) {
    const action = e.target.dataset.action;
    const policyID = e.target.dataset.policyId;

    if (action === 'edit') {
        openEditPolicyModal(policyID);
    } else if (action === 'delete') {
        if (confirm('⚠️ 정말로 이 정책을 삭제하시겠습니까?')) {
            await deletePolicy(policyID);
        }
    }
}

// 정책 편집 모달 열기
function openEditPolicyModal(policyID) {
    const policy = policies.find(p => p.id === policyID);
    if (!policy) return;

    document.getElementById('editPolicyName').value = policy.policy_name;
    document.getElementById('editPolicyData').value = JSON.stringify(JSON.parse(policy.policy_data), null, 2);
    document.getElementById('editPolicyStatus').value = policy.status;
    document.getElementById('editPolicyForm').dataset.policyId = policyID;

    const modal = document.getElementById('editPolicyModal');
    modal.style.display = 'flex';
    modal.classList.add('active');
}

// 정책 생성
async function createPolicy(e) {
    e.preventDefault();

    const policyName = document.getElementById('policyName').value;
    const policyDataStr = document.getElementById('policyData').value;

    if (!policyName || !policyDataStr) {
        showNotification('모든 필드를 입력해주세요', 'error');
        return;
    }

    try {
        JSON.parse(policyDataStr);
    } catch (error) {
        showNotification('정책 데이터가 유효한 JSON이 아닙니다', 'error');
        return;
    }

    try {
        await postAPI('/api/admin/policies', {
            policy_name: policyName,
            policy_data: policyDataStr
        });

        showNotification('✅ 정책이 생성되었습니다!', 'success');
        document.getElementById('policyForm').reset();
        await loadPolicies();
    } catch (error) {
        console.error('Failed to create policy:', error);
        showNotification('정책 생성 실패', 'error');
    }
}

// 정책 수정
async function updatePolicy() {
    const policyForm = document.getElementById('editPolicyForm');
    const policyID = policyForm.dataset.policyId;
    const policyName = document.getElementById('editPolicyName').value;
    const policyDataStr = document.getElementById('editPolicyData').value;
    const status = document.getElementById('editPolicyStatus').value;

    try {
        JSON.parse(policyDataStr);
    } catch (error) {
        showNotification('정책 데이터가 유효한 JSON이 아닙니다', 'error');
        return;
    }

    try {
        const response = await putAPI(`/api/admin/policies/${policyID}`, {
            policy_name: policyName,
            policy_data: policyDataStr,
            status: status
        });

        if (!response.ok) {
            const errorData = await response.json();
            showNotification('정책 수정 실패: ' + (errorData.message || '알 수 없는 오류'), 'error');
            return;
        }

        showNotification('✅ 정책이 수정되었습니다!', 'success');
        closePolicyModal('editPolicyModal');
        await loadPolicies();
    } catch (error) {
        console.error('Failed to update policy:', error);
        showNotification('정책 수정 실패', 'error');
    }
}

// 정책 삭제
async function deletePolicy(policyID) {
    try {
        await deleteAPI(`/api/admin/policies/${policyID}`);
        showNotification('✅ 정책이 삭제되었습니다!', 'success');
        await loadPolicies();
    } catch (error) {
        console.error('Failed to delete policy:', error);
        showNotification('정책 삭제 실패', 'error');
    }
}

// 이벤트 리스너 설정
function setupPolicyEventListeners() {
    document.getElementById('policyForm').addEventListener('submit', createPolicy);

    const editModal = document.getElementById('editPolicyModal');
    editModal.querySelector('[data-action="close"]').addEventListener('click', () => {
        closePolicyModal('editPolicyModal');
    });
    editModal.querySelector('[data-action="save"]').addEventListener('click', updatePolicy);
    editModal.querySelector('.modal-close').addEventListener('click', () => {
        closePolicyModal('editPolicyModal');
    });

    editModal.addEventListener('click', (e) => {
        if (e.target === editModal) {
            closePolicyModal('editPolicyModal');
        }
    });
}

// 모달 닫기
function closePolicyModal(modalID) {
    const modal = document.getElementById(modalID);
    if (modal) {
        modal.style.display = 'none';
        modal.classList.remove('active');
    }
}

// 유틸리티 함수
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showNotification(message, type = 'info') {
    const existing = document.querySelector('.notification-toast');
    if (existing) existing.remove();

    const notification = document.createElement('div');
    notification.className = 'notification-toast';
    notification.textContent = message;
    
    const colors = {
        success: 'linear-gradient(135deg, #51cf66 0%, #37b24d 100%)',
        error: 'linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%)',
        info: 'linear-gradient(135deg, #4dabf7 0%, #339af0 100%)'
    };
    
    notification.style.cssText = `
        position: fixed;
        top: 24px;
        right: 24px;
        padding: 16px 24px;
        border-radius: 12px;
        color: white;
        background: ${colors[type] || colors.info};
        box-shadow: 0 8px 24px rgba(0,0,0,0.2);
        z-index: 99999;
        font-size: 14px;
        font-weight: 600;
        max-width: 400px;
        animation: slideInRight 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    `;
    
    const style = document.createElement('style');
    style.textContent = `
        @keyframes slideInRight {
            from {
                opacity: 0;
                transform: translateX(100px);
            }
            to {
                opacity: 1;
                transform: translateX(0);
            }
        }
    `;
    document.head.appendChild(style);
    
    document.body.appendChild(notification);

    setTimeout(() => {
        notification.style.animation = 'slideInRight 0.3s cubic-bezier(0.4, 0, 0.2, 1) reverse';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}
