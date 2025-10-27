import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { formatDateTime, escapeHtml } from '../utils.js';
import { PolicyEditor } from '../components/policyEditor.js';

let policies = [];
let createPolicyEditor;
let editPolicyEditor;

function ensurePolicyEditor(type) {
  if (type === 'create') {
    if (!createPolicyEditor || !createPolicyEditor.rootEl?.isConnected) {
      const root = document.querySelector('.policy-editor[data-policy-editor="create"]');
      if (root) {
        createPolicyEditor = new PolicyEditor(root);
      } else {
        createPolicyEditor = undefined;
      }
    }
    return createPolicyEditor;
  }

  if (!editPolicyEditor || !editPolicyEditor.rootEl?.isConnected) {
    const root = document.querySelector('.policy-editor[data-policy-editor="edit"]');
    if (root) {
      editPolicyEditor = new PolicyEditor(root);
    } else {
      editPolicyEditor = undefined;
    }
  }
  return editPolicyEditor;
}

// Ensure editors are initialized once DOM is ready
ensurePolicyEditor('create');
ensurePolicyEditor('edit');

export async function loadPolicies() {
  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/policies`, {
      headers: { 'Authorization': `Bearer ${state.token}` }
    });
    const body = await res.json();
    const tbody = document.getElementById('policies-tbody');
    
    if (!tbody) {
      console.error('policies-tbody element not found');
      return;
    }
    
    if (res.ok && body.status === 'success') {
      policies = body.data || [];
      console.log('Loaded policies:', policies);
      
      if (policies.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-center">등록된 정책이 없습니다.</td></tr>';
      } else {
        const html = policies.map(p => `
          <tr>
            <td><strong>${escapeHtml(p.policy_name)}</strong> <small class="mono" style="color:#777;">(${escapeHtml(p.id)})</small></td>
            <td>
              <details style="cursor:pointer;">
                <summary style="color:#667eea;font-weight:600;">데이터 보기</summary>
                <pre style="background:#f8f9fa;padding:12px;border-radius:8px;margin-top:8px;overflow-x:auto;font-size:12px;">${escapeHtml(JSON.stringify(JSON.parse(p.policy_data), null, 2))}</pre>
              </details>
            </td>
            <td style="font-size:13px;color:#6b7280;">
              생성: ${formatDateTime(p.created_at)}<br/>
              수정: ${formatDateTime(p.updated_at)}
            </td>
            <td>
              <button class="btn btn-sm btn-warning" data-action="edit" data-policy-id="${escapeHtml(p.id)}">✏️ 수정</button>
              <button class="btn btn-sm btn-danger" data-action="delete" data-policy-id="${escapeHtml(p.id)}" data-policy-name="${escapeHtml(p.policy_name)}">🗑️ 삭제</button>
            </td>
          </tr>
        `).join('');
        
        tbody.innerHTML = html;
        
        // 이벤트 리스너 추가
        tbody.querySelectorAll('button[data-action]').forEach(btn => {
          btn.addEventListener('click', async (e) => {
            e.preventDefault();
            const action = btn.dataset.action;
            const policyId = btn.dataset.policyId;
            const policyName = btn.dataset.policyName;
            
            if (action === 'edit') {
              openEditPolicyModal(policyId);
            } else if (action === 'delete') {
              await deletePolicy(policyId, policyName);
            }
          });
        });
        
        console.log('Policy table updated successfully');
      }
    } else {
      tbody.innerHTML = `<tr><td colspan="4" class="text-center">불러오기에 실패했습니다: ${escapeHtml(body.message || '')}</td></tr>`;
    }
  } catch (e) {
    console.error('Failed to load policies:', e);
    const tbody = document.getElementById('policies-tbody');
    if (tbody) tbody.innerHTML = '<tr><td colspan="4" class="text-center">서버 오류</td></tr>';
  }
}

export function openCreatePolicyModal() {
  // 폼 리셋
  const form = document.getElementById('create-policy-form');
  if (form) form.reset();

  const editor = ensurePolicyEditor('create');
  if (editor) {
    if (editor.getMode() === 'json') {
      editor.setMode('form');
    }
    editor.resetFields();
    editor.clearError();
  } else {
    const textarea = document.getElementById('policy_data');
    if (textarea) textarea.value = '';
  }

  openModal(document.getElementById('create-policy-modal'));
}

export async function handleCreatePolicy(e) {
  e.preventDefault();
  const policyName = document.getElementById('policy_name').value.trim();
  let policyDataStr = '';

  if (!policyName) {
    showAlert('모든 필드를 입력해주세요.');
    return;
  }

  try {
    const editor = ensurePolicyEditor('create');
    if (editor) {
      policyDataStr = editor.getJsonString();
      editor.clearError();
    } else {
      const textarea = document.getElementById('policy_data');
      const raw = textarea ? textarea.value.trim() : '';
      if (!raw) {
        showAlert('모든 필드를 입력해주세요.');
        return;
      }
      const parsed = JSON.parse(raw);
      policyDataStr = JSON.stringify(parsed, null, 2);
    }
  } catch (err) {
    const editor = ensurePolicyEditor('create');
    if (editor) {
      editor.displayError(err.message);
    }
    showAlert(err.message || '정책 데이터가 유효한 JSON 형식이 아닙니다.');
    return;
  }

  policyDataStr = policyDataStr.trim();
  if (!policyDataStr) {
    showAlert('정책 데이터를 입력해주세요.');
    return;
  }

  const submitBtn = e.target.querySelector('button[type="submit"]');
  const originalBtnText = submitBtn ? submitBtn.textContent : '';
  const originalBtnDisabled = submitBtn ? submitBtn.disabled : false;
  
  if (submitBtn) { 
    submitBtn.disabled = true; 
    submitBtn.textContent = '생성 중...'; 
  }

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/policies`, {
      method: 'POST',
      headers: { 
        'Authorization': `Bearer ${state.token}`, 
        'Content-Type': 'application/json' 
      },
      body: JSON.stringify({
        policy_name: policyName,
        policy_data: policyDataStr
      }),
      _noGlobalLoading: true
    });
    const body = await res.json();
    
    if (res.ok && body.status === 'success') {
      await loadPolicies();
      const modal = document.getElementById('create-policy-modal');
      if (modal) closeModal(modal);
      e.target.reset();
      
      setTimeout(() => {
        showAlert('정책이 생성되었습니다.', '정책 생성 완료');
      }, 300);
    } else {
      // 실패 시에도 모달 닫고 alert
      const modal = document.getElementById('create-policy-modal');
      if (modal) closeModal(modal);
      e.target.reset();
      
      setTimeout(() => {
        showAlert(body.message || '정책 생성에 실패했습니다.', '정책 생성 실패');
      }, 300);
    }
  } catch (err) {
    console.error('Failed to create policy:', err);
    // 에러 시에도 모달 닫고 alert
    const modal = document.getElementById('create-policy-modal');
    if (modal) closeModal(modal);
    e.target.reset();
    
    setTimeout(() => {
      showAlert('서버 오류가 발생했습니다.', '정책 생성 실패');
    }, 300);
  } finally {
    if (submitBtn) {
      submitBtn.disabled = originalBtnDisabled;
      submitBtn.textContent = originalBtnText;
    }
  }
}

function openEditPolicyModal(policyId) {
  const policy = policies.find(p => p.id === policyId);
  if (!policy) {
    showAlert('정책을 찾을 수 없습니다.', 'error');
    return;
  }

  // 폼에 데이터 채우기
  document.getElementById('edit_policy_id').value = policy.id;
  document.getElementById('edit_policy_name').value = policy.policy_name;
  let prettyPolicyData = policy.policy_data || '{}';
  try {
    const parsed = JSON.parse(policy.policy_data);
    prettyPolicyData = JSON.stringify(parsed, null, 2);
  } catch (err) {
    // 그대로 사용 (폼 모드 변환 불가)
  }

  const editor = ensurePolicyEditor('edit');
  if (editor) {
    editor.clearError();
    if (editor.getMode() === 'json') {
      editor.setMode('form');
    }

    try {
      editor.loadFromJson(prettyPolicyData);
      if (editor.textarea) {
        editor.textarea.value = prettyPolicyData;
      }
    } catch (err) {
      // 폼 모드로 변환할 수 없으면 JSON 모드로 전환
      editor.clearError();
      editor.setMode('json');
      if (editor.textarea) {
        editor.textarea.value = prettyPolicyData;
      }
      setTimeout(() => {
        showAlert('이 정책은 폼 모드로 변환할 수 없어 JSON 모드로 전환되었습니다.', '안내');
      }, 100);
    }
  } else {
    const editTextarea = document.getElementById('edit_policy_data');
    if (editTextarea) {
      editTextarea.value = prettyPolicyData;
    }
  }

  openModal(document.getElementById('edit-policy-modal'));
}

export async function handleEditPolicy(e) {
  e.preventDefault();
  const policyId = document.getElementById('edit_policy_id').value;
  const policyName = document.getElementById('edit_policy_name').value.trim();
  let policyDataStr = '';

  if (!policyName) {
    showAlert('모든 필드를 입력해주세요.');
    return;
  }

  try {
    const editor = ensurePolicyEditor('edit');
    if (editor) {
      policyDataStr = editor.getJsonString();
      editor.clearError();
    } else {
      const textarea = document.getElementById('edit_policy_data');
      const raw = textarea ? textarea.value.trim() : '';
      if (!raw) {
        showAlert('모든 필드를 입력해주세요.');
        return;
      }
      const parsed = JSON.parse(raw);
      policyDataStr = JSON.stringify(parsed, null, 2);
    }
  } catch (err) {
    const editor = ensurePolicyEditor('edit');
    if (editor) {
      editor.displayError(err.message);
    }
    showAlert(err.message || '정책 데이터가 유효한 JSON 형식이 아닙니다.');
    return;
  }

  policyDataStr = policyDataStr.trim();
  if (!policyDataStr) {
    showAlert('정책 데이터를 입력해주세요.');
    return;
  }

  const submitBtn = e.target.querySelector('button[type="submit"]');
  const originalBtnText = submitBtn ? submitBtn.textContent : '';
  const originalBtnDisabled = submitBtn ? submitBtn.disabled : false;
  
  if (submitBtn) { 
    submitBtn.disabled = true; 
    submitBtn.textContent = '수정 중...'; 
  }

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/policies/${policyId}`, {
      method: 'PUT',
      headers: { 
        'Authorization': `Bearer ${state.token}`, 
        'Content-Type': 'application/json' 
      },
      body: JSON.stringify({
        policy_name: policyName,
        policy_data: policyDataStr
      }),
      _noGlobalLoading: true
    });
    const body = await res.json();
    
    if (res.ok && body.status === 'success') {
      await loadPolicies();
      const modal = document.getElementById('edit-policy-modal');
      if (modal) closeModal(modal);
      
      setTimeout(() => {
        showAlert('정책이 수정되었습니다.', '정책 수정 완료');
      }, 300);
    } else {
      // 실패 시에도 모달 닫고 alert
      const modal = document.getElementById('edit-policy-modal');
      if (modal) closeModal(modal);
      
      setTimeout(() => {
        showAlert(body.message || '정책 수정에 실패했습니다.', '정책 수정 실패');
      }, 300);
    }
  } catch (err) {
    console.error('Failed to update policy:', err);
    // 에러 시에도 모달 닫고 alert
    const modal = document.getElementById('edit-policy-modal');
    if (modal) closeModal(modal);
    
    setTimeout(() => {
      showAlert('서버 오류가 발생했습니다.', '정책 수정 실패');
    }, 300);
  } finally {
    if (submitBtn) {
      submitBtn.disabled = originalBtnDisabled;
      submitBtn.textContent = originalBtnText;
    }
  }
}

async function deletePolicy(policyId, policyName) {
  const confirmed = await showConfirm(
    `정책 "${policyName}"을(를) 삭제하시겠습니까?`,
    '이 작업은 되돌릴 수 없습니다.'
  );
  
  if (!confirmed) return;

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/policies/${policyId}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${state.token}` },
      _noGlobalLoading: true
    });
    const body = await res.json();
    
    if (res.ok && body.status === 'success') {
      await loadPolicies();
      showAlert('정책이 삭제되었습니다.', 'success');
    } else {
      showAlert(body.message || '정책 삭제에 실패했습니다.', 'error');
    }
  } catch (err) {
    console.error('Failed to delete policy:', err);
    showAlert('서버 오류가 발생했습니다.', 'error');
  }
}

// 전역 함수로 노출
window.openCreatePolicyModal = openCreatePolicyModal;
window.loadPolicies = loadPolicies;
