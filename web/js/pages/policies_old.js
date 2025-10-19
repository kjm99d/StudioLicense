// 정책 관리 페이지
import { getAPI, postAPI, putAPI, deleteAPI } from '../api.js';
import { formatDate } from '../utils.js';

let policies = [];

// 정책 페이지 초기화
export async function initPoliciesPage() {
    console.log('🔧 Initializing policies page...');
    const container = document.getElementById('page-content');
    console.log('📦 page-content container:', container);
    
    if (!container) {
        console.error('❌ page-content element not found!');
        return;
    }
    
    container.innerHTML = `
        <style>
        .policies-container {
            padding: 20px;
        }
        .policy-form {
            background: #f5f5f5;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
            font-size: 14px;
        }
        .form-group input, .form-group textarea, .form-group select {
            width: 100%;
            padding: 8px 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 13px;
            box-sizing: border-box;
        }
        .form-group input:focus, .form-group textarea:focus, .form-group select:focus {
            outline: none;
            border-color: #007bff;
            box-shadow: 0 0 0 2px rgba(0,123,255,0.1);
        }
        .policy-item {
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 15px;
            background: white;
        }
        .policy-actions {
            margin-top: 10px;
        }
        .btn {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            margin-right: 5px;
            font-size: 13px;
            font-weight: 500;
            transition: all 0.2s ease;
        }
        .btn:hover {
            opacity: 0.9;
        }
        .btn:active {
            transform: translateY(1px);
        }
        .btn-primary {
            background: #007bff;
            color: white;
        }
        .btn-primary:hover {
            background: #0056b3;
        }
        .btn-secondary {
            background: #6c757d;
            color: white;
        }
        .btn-secondary:hover {
            background: #545b62;
        }
        .btn-danger {
            background: #dc3545;
            color: white;
        }
        .btn-danger:hover {
            background: #c82333;
        }
        .btn-sm {
            padding: 4px 8px;
            font-size: 12px;
        }
        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0,0,0,0.5);
            overflow-y: auto;
        }
        .modal.active {
            display: block !important;
        }
        .modal-content {
            background-color: white;
            margin: 50px auto;
            padding: 0;
            border-radius: 8px;
            width: 90%;
            max-width: 500px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }
        .modal-header {
            padding: 15px 20px;
            border-bottom: 1px solid #ddd;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .modal-header h3 {
            margin: 0;
            font-size: 18px;
        }
        .modal-body {
            padding: 20px;
            max-height: 70vh;
            overflow-y: auto;
        }
        .modal-footer {
            padding: 15px 20px;
            border-top: 1px solid #ddd;
            text-align: right;
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }
        .modal-close {
            background: none;
            border: none;
            font-size: 20px;
            cursor: pointer;
            color: #666;
            padding: 0;
        }
        .modal-close:hover {
            color: #000;
        }
        .status-badge {
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 12px;
        }
        .status-badge.active {
            background: #d4edda;
            color: #155724;
        }
        .status-badge.inactive {
            background: #f8d7da;
            color: #721c24;
        }
        .policy-data-preview pre {
            background: #f8f9fa;
            padding: 10px;
            border-radius: 4px;
            overflow-x: auto;
            font-size: 12px;
        }
        </style>
        
        <div class="policies-container">
            <div class="page-header">
                <h2>정책 관리</h2>
                <p class="subtitle">시스템 정책을 관리합니다</p>
            </div>

            <div class="policies-content">
                <div class="policy-form-section">
                    <h3>정책 생성</h3>
                    <form id="policyForm" class="policy-form">
                        <div class="form-group">
                            <label for="policyName">정책명 *</label>
                            <input type="text" id="policyName" placeholder="정책명 (예: 기본정책, 프로정책)" required />
                        </div>

                        <div class="form-group">
                            <label for="policyData">정책 데이터 (JSON) *</label>
                            <textarea id="policyData" placeholder='{"key": "value"}' rows="6" required></textarea>
                        </div>

                        <button type="submit" class="btn btn-primary">정책 생성</button>
                    </form>
                </div>

                <div class="policy-list-section">
                    <h3>정책 목록</h3>
                    <div id="policiesList" class="policies-list">
                        <p class="loading">정책을 로드 중...</p>
                    </div>
                </div>
            </div>
        </div>

        <!-- 정책 수정 모달 -->
        <div id="editPolicyModal" class="modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h3>정책 수정</h3>
                    <button type="button" class="modal-close">&times;</button>
                </div>
                <div class="modal-body">
                    <form id="editPolicyForm">
                        <div class="form-group">
                            <label for="editPolicyName">정책명</label>
                            <input type="text" id="editPolicyName" />
                        </div>

                        <div class="form-group">
                            <label for="editPolicyData">정책 데이터 (JSON)</label>
                            <textarea id="editPolicyData" rows="6"></textarea>
                        </div>

                        <div class="form-group">
                            <label for="editPolicyStatus">상태</label>
                            <select id="editPolicyStatus">
                                <option value="active">활성</option>
                                <option value="inactive">비활성</option>
                            </select>
                        </div>
                    </form>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-action="close">취소</button>
                    <button type="button" class="btn btn-primary" data-action="save">저장</button>
                </div>
            </div>
        </div>
    `;

    // 이벤트 리스너 설정
    setupPolicyEventListeners();
    
    // 정책 목록 로드
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
        container.innerHTML = '<p class="no-data">등록된 정책이 없습니다</p>';
        return;
    }

    container.innerHTML = policies.map(policy => `
        <div class="policy-item">
            <div class="policy-info">
                <h4>${escapeHtml(policy.policy_name)}</h4>
                <p class="policy-id">ID: ${policy.id}</p>
                <div class="policy-status">
                    <span class="status-badge ${policy.status}">${policy.status === 'active' ? '활성' : '비활성'}</span>
                </div>
                <div class="policy-data-preview">
                    <pre>${escapeHtml(JSON.stringify(JSON.parse(policy.policy_data), null, 2))}</pre>
                </div>
                <p class="policy-meta">
                    생성: ${formatDate(policy.created_at)} | 수정: ${formatDate(policy.updated_at)}
                </p>
            </div>
            <div class="policy-actions">
                <button class="btn btn-sm btn-secondary" data-action="edit" data-policy-id="${policy.id}">수정</button>
                <button class="btn btn-sm btn-danger" data-action="delete" data-policy-id="${policy.id}">삭제</button>
            </div>
        </div>
    `).join('');

    // 액션 버튼 리스너 설정
    container.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', handlePolicyAction);
    });
}

// 정책 액션 처리
async function handlePolicyAction(e) {
    const action = e.target.dataset.action;
    const policyID = e.target.dataset.policyId;

    console.log('정책 액션:', action, '정책 ID:', policyID);

    if (action === 'edit') {
        console.log('정책 수정 모달 열기 시도...');
        openEditPolicyModal(policyID);
    } else if (action === 'delete') {
        if (confirm('정책을 삭제하시겠습니까?')) {
            await deletePolicy(policyID);
        }
    }
}

// 정책 편집 모달 열기
async function openEditPolicyModal(policyID) {
    console.log('정책 편집 모달 열기:', policyID);
    const policy = policies.find(p => p.id === policyID);
    console.log('찾은 정책:', policy);
    
    if (!policy) {
        console.error('정책을 찾을 수 없습니다:', policyID);
        return;
    }

    // 모달 필드 설정
    const nameField = document.getElementById('editPolicyName');
    const dataField = document.getElementById('editPolicyData');
    const statusField = document.getElementById('editPolicyStatus');
    const formField = document.getElementById('editPolicyForm');
    
    console.log('모달 필드들:', { nameField, dataField, statusField, formField });
    
    if (!nameField || !dataField || !statusField || !formField) {
        console.error('모달 필드를 찾을 수 없습니다');
        return;
    }

    nameField.value = policy.policy_name;
    dataField.value = JSON.stringify(JSON.parse(policy.policy_data), null, 2);
    statusField.value = policy.status;
    formField.dataset.policyId = policyID;

    // 모달 열기
    const modal = document.getElementById('editPolicyModal');
    console.log('모달 요소:', modal);
    
    if (modal) {
        modal.style.display = 'block';
        modal.classList.add('active');
        console.log('모달이 열렸습니다');
    } else {
        console.error('editPolicyModal을 찾을 수 없습니다');
    }
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
        // JSON 유효성 검사
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

        showNotification('정책이 생성되었습니다', 'success');
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

    console.log('정책 수정 시도:', { policyID, policyName, status });

    try {
        // JSON 유효성 검사
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

        console.log('정책 수정 응답:', response);
        
        if (!response.ok) {
            const errorData = await response.json();
            console.error('정책 수정 오류:', errorData);
            showNotification('정책 수정 실패: ' + (errorData.message || '알 수 없는 오류'), 'error');
            return;
        }

        showNotification('정책이 수정되었습니다', 'success');
        closePolicyModal('editPolicyModal');
        await loadPolicies();
    } catch (error) {
        console.error('Failed to update policy:', error);
        showNotification('정책 수정 실패: ' + error.message, 'error');
    }
}

// 정책 삭제
async function deletePolicy(policyID) {
    try {
        await deleteAPI(`/api/admin/policies/${policyID}`);
        showNotification('정책이 삭제되었습니다', 'success');
        await loadPolicies();
    } catch (error) {
        console.error('Failed to delete policy:', error);
        showNotification('정책 삭제 실패', 'error');
    }
}

// 정책 이벤트 리스너 설정
function setupPolicyEventListeners() {
    console.log('⚙️ Setting up policy event listeners...');

    // 정책 생성 폼
    document.getElementById('policyForm').addEventListener('submit', createPolicy);

    // 정책 편집 모달
    const editModal = document.getElementById('editPolicyModal');
    editModal.querySelector('[data-action="close"]').addEventListener('click', () => {
        closePolicyModal('editPolicyModal');
    });
    editModal.querySelector('[data-action="save"]').addEventListener('click', updatePolicy);
    editModal.querySelector('.modal-close').addEventListener('click', () => {
        closePolicyModal('editPolicyModal');
    });

    // 모달 닫기 (외부 클릭)
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
    // 기존 알림 제거
    const existing = document.querySelector('.notification-container');
    if (existing) {
        existing.remove();
    }

    // 새로운 알림 생성
    const container = document.createElement('div');
    container.className = 'notification-container';
    container.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 10000;
    `;

    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        padding: 12px 16px;
        border-radius: 4px;
        color: white;
        background: ${type === 'error' ? '#dc3545' : type === 'success' ? '#28a745' : '#17a2b8'};
        box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        font-size: 14px;
        max-width: 300px;
        word-wrap: break-word;
    `;

    container.appendChild(notification);
    document.body.appendChild(container);

    setTimeout(() => {
        container.remove();
    }, 3000);
}