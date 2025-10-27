import { state } from '../state.js';
import { apiFetch, API_BASE_URL } from '../api.js';
import { openModal, closeModal, showAlert, showConfirm } from '../modals.js';
import { escapeHtml, formatDateTime } from '../utils.js';

const filesState = {
  items: [],
  page: 1,
  totalPages: 0,
  totalCount: 0,
  search: '',
  pageSize: 20,
  initialized: false,
};

export async function loadFiles(page = 1, options = {}) {
  if (!state.token) return;
  if (options.search !== undefined) {
    filesState.search = options.search.trim();
  }
  const params = new URLSearchParams();
  params.set('page', page);
  params.set('limit', filesState.pageSize);
  if (filesState.search) {
    params.set('q', filesState.search);
  }

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/files?${params.toString()}`, {
      headers: { Authorization: `Bearer ${state.token}` },
    });
    const body = await res.json();
    if (res.ok && body.status === 'success') {
      filesState.items = body.data || [];
      filesState.page = body.meta?.page ?? page;
      filesState.totalPages = body.meta?.total_pages ?? 0;
      filesState.totalCount = body.meta?.total_count ?? filesState.items.length;
      renderFilesTable();
      renderFilesPagination();
    } else {
      await showAlert(body.message || '파일 목록을 불러오지 못했습니다.', '파일 목록 오류');
    }
  } catch (err) {
    console.error('Failed to load files:', err);
    await showAlert('파일 목록을 불러오지 못했습니다.', '파일 목록 오류');
  }
}

export function openUploadFileModal() {
  const form = document.getElementById('upload-file-form');
  if (form) {
    form.reset();
  }
  const modal = document.getElementById('upload-file-modal');
  if (modal) openModal(modal);
}

export async function handleFileUpload(e) {
  e.preventDefault();
  const form = e.target;
  const fileInput = form.querySelector('#upload_file_input');
  if (!fileInput || !fileInput.files || fileInput.files.length === 0) {
    await showAlert('업로드할 파일을 선택해주세요.', '업로드 오류');
    return;
  }

  const submitBtn = form.querySelector('button[type="submit"]');
  const originalText = submitBtn ? submitBtn.textContent : '';
  if (submitBtn) {
    submitBtn.disabled = true;
    submitBtn.textContent = '업로드 중...';
  }

  const formData = new FormData(form);

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/files`, {
      method: 'POST',
      body: formData,
      headers: { Authorization: `Bearer ${state.token}` },
      _noGlobalLoading: true,
    });
    const body = await res.json();
    if (res.ok && body.status === 'success') {
      const modal = document.getElementById('upload-file-modal');
      if (modal) closeModal(modal);
      await showAlert('파일이 업로드되었습니다.', '업로드 완료');
      await loadFiles(1);
    } else {
      await showAlert(body.message || '파일 업로드에 실패했습니다.', '업로드 오류');
    }
  } catch (err) {
    console.error('Failed to upload file:', err);
    await showAlert('파일 업로드 중 오류가 발생했습니다.', '업로드 오류');
  } finally {
    if (submitBtn) {
      submitBtn.disabled = false;
      submitBtn.textContent = originalText;
    }
  }
}

export function initFilesPage() {
  if (filesState.initialized) return;
  filesState.initialized = true;

  const form = document.getElementById('upload-file-form');
  if (form) {
    form.addEventListener('submit', handleFileUpload);
  }

  const searchForm = document.getElementById('file-search-form');
  if (searchForm) {
    searchForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const searchInput = document.getElementById('file-search-input');
      const value = searchInput ? searchInput.value : '';
      await loadFiles(1, { search: value || '' });
    });
  }
}

function renderFilesTable() {
  const tbody = document.getElementById('files-tbody');
  if (!tbody) return;

  if (!filesState.items.length) {
    tbody.innerHTML = `<tr><td colspan="5" class="text-center">등록된 파일이 없습니다.</td></tr>`;
    return;
  }

  const rows = filesState.items.map((file) => {
    const size = formatFileSize(file.file_size);
    const uploader = file.uploaded_username || '-';
    const created = formatDateTime(file.created_at);
    const description = file.description ? `<div class="file-description">${escapeHtml(file.description)}</div>` : '';
    return `
      <tr data-file-id="${escapeHtml(file.id)}">
        <td>
          <div class="file-name">
            <strong>${escapeHtml(file.original_name)}</strong>
            <small class="mono" style="color:#6b7280;">${escapeHtml(file.mime_type || 'unknown')}</small>
          </div>
          ${description}
        </td>
        <td>${escapeHtml(uploader)}</td>
        <td>${size}</td>
        <td>${created}</td>
        <td class="file-actions">
          <button class="btn btn-sm btn-outline blue lighten-1" data-action="download">⬇ 다운로드</button>
          <button class="btn btn-sm btn-danger" data-action="delete">🗑 삭제</button>
        </td>
      </tr>
    `;
  }).join('');

  tbody.innerHTML = rows;

  tbody.querySelectorAll('tr').forEach((row, index) => {
    const file = filesState.items[index];
    row.querySelectorAll('button[data-action]').forEach((btn) => {
      const action = btn.dataset.action;
      if (action === 'download') {
        btn.addEventListener('click', async (e) => {
          e.preventDefault();
          await downloadFile(file);
        });
      } else if (action === 'delete') {
        btn.addEventListener('click', async (e) => {
          e.preventDefault();
          await deleteFile(file);
        });
      }
    });
  });
}

function renderFilesPagination() {
  const container = document.getElementById('files-pagination');
  if (!container) return;

  if (filesState.totalPages <= 1) {
    container.innerHTML = '';
    return;
  }

  const pages = [];
  for (let i = 1; i <= filesState.totalPages; i++) {
    pages.push(`<button class="btn btn-sm ${filesState.page === i ? 'blue' : 'grey lighten-2'}" data-page="${i}">${i}</button>`);
  }
  container.innerHTML = pages.join('');

  container.querySelectorAll('button[data-page]').forEach((btn) => {
    btn.addEventListener('click', async (e) => {
      e.preventDefault();
      const targetPage = parseInt(btn.dataset.page, 10);
      if (!Number.isNaN(targetPage) && targetPage !== filesState.page) {
        await loadFiles(targetPage);
      }
    });
  });
}

async function deleteFile(file) {
  const confirmed = await showConfirm(`"${file.original_name}" 파일을 삭제하시겠습니까?`, '파일 삭제');
  if (!confirmed) return;

  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/files/${file.id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${state.token}`, 'Content-Type': 'application/json' },
      _noGlobalLoading: true,
    });
    const body = await res.json();
    if (res.ok && body.status === 'success') {
      await showAlert('파일이 삭제되었습니다.', '삭제 완료');
      await loadFiles(filesState.page);
    } else {
      await showAlert(body.message || '파일 삭제에 실패했습니다.', '삭제 오류');
    }
  } catch (err) {
    console.error('Failed to delete file:', err);
    await showAlert('파일 삭제 중 오류가 발생했습니다.', '삭제 오류');
  }
}

async function downloadFile(file) {
  try {
    const res = await apiFetch(`${API_BASE_URL}/api/admin/files/${file.id}?download=1`, {
      headers: { Authorization: `Bearer ${state.token}` },
      _noGlobalLoading: true,
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({ message: '다운로드에 실패했습니다.' }));
      await showAlert(body.message || '다운로드에 실패했습니다.', '다운로드 오류');
      return;
    }
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = file.original_name;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  } catch (err) {
    console.error('Failed to download file:', err);
    await showAlert('다운로드 중 오류가 발생했습니다.', '다운로드 오류');
  }
}

function formatFileSize(size) {
  if (!Number.isFinite(size) || size < 0) return '-';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let index = 0;
  let value = size;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}
