// 제품 관리 페이지

import { state } from '../state.js';
import { showAlert, openModal, closeModal, showConfirm } from '../modals.js';
import { renderStatusBadge } from '../ui.js';
import { formatDate, escapeHtml } from '../utils.js';
import { apiFetch, API_BASE_URL } from '../api.js';

const FILE_FETCH_LIMIT = 200;
let currentProductFilesProductId = null;
let currentProductFilesProductName = '';
let currentProductFileMappings = [];
let availableFileAssets = [];
let currentProductFilesFilter = '';
let currentProductFilesRoot = null;
let activeProductFilesRow = null;
let activeProductFilesButton = null;

function getProductFilesElement(role) {
    if (!currentProductFilesRoot) return null;
    return currentProductFilesRoot.querySelector(`[data-role="${role}"]`);
}

function updateManageButtonState(productId, isExpanded) {
    if (activeProductFilesButton) {
        activeProductFilesButton.classList.remove('is-active');
        activeProductFilesButton.setAttribute('aria-expanded', 'false');
        activeProductFilesButton = null;
    }

    if (!isExpanded) return;

    const button = document.querySelector(`button[data-role="product-manage-files"][data-product-id="${productId}"]`);
    if (button) {
        button.classList.add('is-active');
        button.setAttribute('aria-expanded', 'true');
        activeProductFilesButton = button;
    }
}

function collapseProductFilesPanel() {
    if (activeProductFilesRow?.parentElement) {
        activeProductFilesRow.remove();
    }
    if (activeProductFilesButton) {
        activeProductFilesButton.classList.remove('is-active');
        activeProductFilesButton.setAttribute('aria-expanded', 'false');
        activeProductFilesButton = null;
    }
    currentProductFilesProductId = null;
    currentProductFilesProductName = '';
    currentProductFilesFilter = '';
    currentProductFilesRoot = null;
    activeProductFilesRow = null;
}

function buildProductFilesPanelMarkup(productName = '') {
    const safeName = escapeHtml(productName || '');

    return `
        <div class="product-files-panel" data-role="product-files-root">
            <div class="product-files-panel-header">
                <div>
                    <span class="panel-subtitle">선택한 제품</span>
                    <h3 class="product-files-target-name" data-role="product-files-target-name">${safeName || '제품을 선택하세요'}</h3>
                </div>
                <button type="button" class="product-files-close-btn" aria-label="닫기" data-role="product-files-close">
                    <span aria-hidden="true">&times;</span>
                </button>
            </div>
            <div class="product-files-top">
                <form class="product-file-form" data-role="product-file-form">
                    <div class="product-file-form-header">
                        <div class="form-header-row">
                            <h4 data-role="product-file-form-title">새 파일 연결</h4>
                            <span class="form-mode-chip is-create" data-role="product-file-form-mode">새 연결</span>
                        </div>
                        <p class="form-helper-text" data-role="product-file-file-helper">제품에 연결할 파일을 선택하세요.</p>
                    </div>
                    <input type="hidden" data-role="product-file-id">
                    <div class="product-file-form-grid">
                        <div class="form-group form-group--full">
                            <label>연결할 파일</label>
                            <select class="browser-default" data-role="product-file-select"></select>
                        </div>
                        <div class="form-group">
                            <label>표시명 *</label>
                            <input type="text" data-role="product-file-label" required>
                        </div>
                        <div class="form-group form-group--auto">
                            <label>정렬 순서</label>
                            <input type="number" data-role="product-file-sort" value="0" min="0">
                            <p class="field-helper-text">숫자가 낮을수록 목록 상단에 노출됩니다.</p>
                        </div>
                        <div class="form-group form-group--full">
                            <label>설명</label>
                            <textarea rows="3" data-role="product-file-description" placeholder="이 파일에 대한 간단한 설명을 입력하세요."></textarea>
                        </div>
                        <div class="form-group form-group--full">
                            <label>외부 다운로드 URL</label>
                            <input type="url" data-role="product-file-delivery-url" placeholder="https://example.com/download" autocomplete="off">
                            <p class="field-helper-text">CDN 또는 외부 링크를 통해 파일을 전달할 때만 입력하세요.</p>
                        </div>
                    </div>
                    <div class="product-file-form-actions">
                        <button type="button" class="btn grey lighten-1" data-role="product-file-reset">새 연결</button>
                        <button type="submit" class="btn blue" data-role="product-file-submit">연결 추가</button>
                    </div>
                </form>
            </div>
            <div class="product-files-table-wrapper" data-role="product-files-table-wrapper">
                 
                 <div class="product-files-list-header">
                     <div>
                         <h4>연결된 파일</h4>
                         <p class="list-helper-text">사용자에게 노출되는 파일과 외부 다운로드 링크를 확인하고 관리하세요.</p>
                     </div>
                     <span class="list-count-chip" data-role="product-files-list-count">-</span>
                 </div>
                <div class="product-file-toolbar">
                     <div class="product-file-search">
                         <div class="search-input-wrapper">
                             <span class="search-icon" aria-hidden="true">
                                 <svg width="16" height="16" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                                     <circle cx="9" cy="9" r="6" stroke="currentColor" stroke-width="1.5"></circle>
                                     <path d="M13.5 13.5L17 17" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"></path>
                                 </svg>
                             </span>
                             <input type="search" placeholder="파일명, 표시명, 설명 검색" autocomplete="off" data-role="product-file-search">
                         </div>
                     </div>
                 </div>
                 <table class="striped highlight product-files-table">
                     <thead>
                         <tr>
                             <th>표시명</th>
                             <th>파일</th>
                             <th>설명</th>
                            <th>정렬</th>
                            <th>링크</th>
                            <th>외부 링크</th>
                            <th>작업</th>
                         </tr>
                     </thead>
                     <tbody data-role="product-files-tbody">
                         <tr>
                             <td colspan="7" class="text-center">연결된 파일이 없습니다.</td>
                         </tr>
                     </tbody>
                 </table>
             </div>
        </div>
    `;
}

function attachProductFilesPanelEvents() {
    if (!currentProductFilesRoot) return;

    const form = getProductFilesElement('product-file-form');
    if (form) {
        form.addEventListener('submit', handleProductFileSubmit);
    }

    const resetBtn = getProductFilesElement('product-file-reset');
    if (resetBtn) {
        resetBtn.addEventListener('click', (event) => {
            event.preventDefault();
            resetProductFileForm();
        });
    }

    const closeBtn = getProductFilesElement('product-files-close');
    if (closeBtn) {
        closeBtn.addEventListener('click', () => collapseProductFilesPanel());
    }

    const searchInput = getProductFilesElement('product-file-search');
    if (searchInput) {
        searchInput.addEventListener('input', handleProductFilesSearch);
    }

    const tableWrapper = getProductFilesElement('product-files-table-wrapper');
    if (tableWrapper) {
        tableWrapper.removeEventListener('click', handleProductFilesTableClick);
        tableWrapper.addEventListener('click', handleProductFilesTableClick);
    }
}

async function loadProducts() {
    try {
        console.log('Loading products...', state.token);
        const response = await fetch('/api/admin/products', {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${state.token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        console.log('Products result:', result);
        
        if (result.status === 'success') {
            renderProductsTable(result.data || []);
        } else {
            showAlert('제품 목록 조회 실패', 'error');
        }
    } catch (error) {
        console.error('Error loading products:', error);
        showAlert('제품 목록을 불러올 수 없습니다.', 'error');
    }
}

function renderProductsTable(products) {
    const tbody = document.querySelector('#products-table');
    if (!tbody) return;

    collapseProductFilesPanel();
    tbody.innerHTML = '';

    if (!products || products.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; padding: 20px;">제품이 없습니다.</td></tr>';
        return;
    }

    products.forEach((product) => {
        const row = document.createElement('tr');
        const statusBadge = renderStatusBadge(product.status);
        const createdDate = formatDate(product.created_at);
        const productId = String(product.id || '');
        const productName = product.name || '';

        row.dataset.productId = productId;

        row.innerHTML = `
            <td>${escapeHtml(productName)}</td>
            <td>${product.description ? escapeHtml(product.description) : '-'}</td>
            <td>${statusBadge}</td>
            <td>${createdDate}</td>
            <td style="white-space: nowrap;">
                <button
                    class="btn btn-sm grey lighten-2"
                    data-role="product-manage-files"
                    data-product-id="${escapeHtml(productId)}"
                    data-product-name="${escapeHtml(productName)}"
                    aria-expanded="false"
                    onclick="manageProductFiles(this.dataset.productId, this.dataset.productName)">
                    파일 관리
                </button>
                <button class="btn btn-sm btn-primary" onclick="editProduct('${product.id}')">수정</button>
                <button class="btn btn-sm btn-danger" onclick="deleteProductConfirm('${product.id}', '${product.name}')">삭제</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

function editProduct(productId) {
    window.editingProductId = productId;
    showProductModal(productId);
}

function deleteProductConfirm(productId, productName) {
    if (confirm(`"${productName}" 제품을 삭제하시겠습니까?`)) {
        deleteProduct(productId);
    }
}

async function deleteProduct(productId) {
    try {
        const response = await fetch(`/api/admin/products/?id=${productId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${state.token}`,
                'Content-Type': 'application/json'
            }
        });

        const result = await response.json();

        if (response.ok) {
            showAlert('제품이 삭제되었습니다.', 'success');
            loadProducts();
        } else {
            showAlert(result.message || '삭제에 실패했습니다.', 'error');
        }
    } catch (error) {
        console.error('Error deleting product:', error);
        showAlert('삭제 중 오류가 발생했습니다.', 'error');
    }
}

function showProductModal(productId = null) {
    const modal = document.getElementById('product-modal');
    if (!modal) return;

    const titleEl = modal.querySelector('#product-modal-title');
    const nameInput = modal.querySelector('#product_name_input');
    const descriptionInput = modal.querySelector('#product_description');
    const submitBtn = modal.querySelector('button[type="submit"]');

    if (productId) {
        // 수정 모드
        titleEl.textContent = '제품 수정';
        if (submitBtn) submitBtn.textContent = '수정';
        loadProductDetail(productId, nameInput, descriptionInput);
    } else {
        // 생성 모드
        titleEl.textContent = '새 제품 생성';
        if (submitBtn) submitBtn.textContent = '생성';
        nameInput.value = '';
        descriptionInput.value = '';
        window.editingProductId = null;
    }

    if (document.activeElement instanceof HTMLElement) {
        modal._triggerElement = document.activeElement;
    }

    openModal(modal);
}

async function loadProductDetail(productId, nameInput, descriptionInput) {
    try {
        const response = await fetch(`/api/admin/products/?id=${productId}`, {
            headers: {
                'Authorization': `Bearer ${state.token}`
            }
        });

        const result = await response.json();
        if (result.status === 'success' && result.data) {
            const product = result.data;
            nameInput.value = product.name;
            descriptionInput.value = product.description || '';
        }
    } catch (error) {
        console.error('Error loading product detail:', error);
    }
}

async function handleCreateProduct(event) {
    event.preventDefault();

    const nameInput = document.querySelector('#product_name_input');
    const descriptionInput = document.querySelector('#product_description');
    const submitBtn = document.querySelector('#product-form button[type="submit"]');

    if (!nameInput.value.trim()) {
        showAlert('제품명을 입력하세요.', 'error');
        return;
    }

    const submitText = submitBtn.textContent;
    submitBtn.textContent = '처리 중...';
    submitBtn.disabled = true;

    try {
        const isUpdate = window.editingProductId !== null && window.editingProductId !== undefined;
        const method = isUpdate ? 'PUT' : 'POST';
        const url = isUpdate 
            ? `/api/admin/products/?id=${window.editingProductId}`
            : '/api/admin/products';

        const payload = {
            name: nameInput.value.trim(),
            description: descriptionInput.value.trim(),
            status: 'active'
        };

        const response = await fetch(url, {
            method: method,
            headers: {
                'Authorization': `Bearer ${state.token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        });

        const result = await response.json();

        if (response.ok) {
            showAlert(isUpdate ? '제품이 수정되었습니다.' : '제품이 생성되었습니다.', 'success');
            closeProductModal();
            loadProducts();
        } else {
            showAlert(result.message || '처리에 실패했습니다.', 'error');
        }
    } catch (error) {
        console.error('Error creating/updating product:', error);
        showAlert('오류가 발생했습니다.', 'error');
    } finally {
        submitBtn.textContent = submitText;
        submitBtn.disabled = false;
    }
}

function closeProductModal() {
    const modal = document.getElementById('product-modal');
    if (modal) {
        closeModal(modal);
    }
    window.editingProductId = null;
}

function formatFileSize(bytes) {
    if (bytes === undefined || bytes === null || Number.isNaN(bytes)) return '-';
    const units = ['KB', 'MB', 'GB', 'TB'];
    let value = Number(bytes);
    if (Number.isNaN(value)) return '-';
    if (value < 1024) return `${value} B`;
    let unitIndex = -1;
    do {
        value /= 1024;
        unitIndex += 1;
    } while (value >= 1024 && unitIndex < units.length - 1);
    const digits = value >= 10 ? 0 : 1;
    return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

function ensureOptionForFile(fileId, file, select = getProductFilesElement('product-file-select')) {
    if (!select || !fileId) return;
    const exists = Array.from(select.options).some((opt) => opt.value === fileId);
    if (exists) return;

    const label = file?.original_name ? `${file.original_name} (등록됨)` : fileId;
    const option = document.createElement('option');
    option.value = fileId;
    option.textContent = label;
    option.dataset.injected = '1';
    select.appendChild(option);
}

function ensureMappingsFilesExistInSelect(select = getProductFilesElement('product-file-select')) {
    currentProductFileMappings.forEach((mapping) => {
        ensureOptionForFile(mapping.file_id, mapping.file, select);
    });
}

function populateProductFileSelect() {
    const select = getProductFilesElement('product-file-select');
    if (!select) return;

    const previousValue = select.value;
    const wasDisabled = select.disabled;

    select.innerHTML = '';
    const placeholder = document.createElement('option');
    placeholder.value = '';
    placeholder.textContent = availableFileAssets.length ? '연결할 파일을 선택하세요' : '등록된 파일이 없습니다';
    placeholder.disabled = availableFileAssets.length === 0;
    placeholder.selected = true;
    select.appendChild(placeholder);

    availableFileAssets.forEach((file) => {
        const option = document.createElement('option');
        option.value = file.id;
        option.textContent = `${file.original_name} (${formatFileSize(file.file_size)})`;
        option.title = `${file.original_name}${file.description ? ` · ${file.description}` : ''}`;
        select.appendChild(option);
    });

    ensureMappingsFilesExistInSelect(select);

    if (previousValue && Array.from(select.options).some((opt) => opt.value === previousValue)) {
        select.value = previousValue;
    }

    select.disabled = wasDisabled;
}

function getFilteredProductFileMappings() {
    const keyword = currentProductFilesFilter.trim().toLowerCase();
    if (!keyword) {
        return [...currentProductFileMappings];
    }

    return currentProductFileMappings.filter((mapping) => {
        const candidates = [
            mapping.label || '',
            mapping.description || '',
            mapping.file_id || '',
            mapping.file?.original_name || '',
            mapping.file?.mime_type || ''
        ];

        return candidates.some((value) => value.toLowerCase().includes(keyword));
    });
}

function renderProductFilesSummary() {
    if (!currentProductFilesRoot) return;

    const total = currentProductFileMappings.length;
    const listCount = currentProductFilesRoot.querySelector('[data-role="product-files-list-count"]');
    if (listCount) listCount.textContent = total;
}

async function downloadProductFileAsset(fileId, fileName) {
    if (!fileId) {
        await showAlert('다운로드 정보가 올바르지 않습니다.', '다운로드 오류');
        return;
    }

    if (!state.token) {
        await showAlert('인증이 만료되었습니다. 다시 로그인해 주세요.', '다운로드 오류');
        return;
    }

    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/files/${encodeURIComponent(fileId)}?download=1`, {
            headers: { Authorization: `Bearer ${state.token}` },
            _noGlobalLoading: true,
        });

        if (!response.ok) {
            let message = '파일 다운로드에 실패했습니다.';
            try {
                const body = await response.json();
                message = body?.message || body?.error || message;
            } catch (error) {
                // ignore JSON parse error, use default message
            }
            await showAlert(message, '다운로드 오류');
            return;
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = fileName || 'download';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);
    } catch (error) {
        console.error('Failed to download product file asset:', error);
        await showAlert('다운로드 중 오류가 발생했습니다.', '다운로드 오류');
    }
}

async function handleProductFilesTableClick(event) {
    const downloadBtn = event.target.closest('[data-action="product-file-download"]');
    if (downloadBtn && currentProductFilesRoot?.contains(downloadBtn)) {
        event.preventDefault();
        const { fileId, fileName } = downloadBtn.dataset;
        await downloadProductFileAsset(fileId, fileName);
    }
}

function renderProductFilesLoading() {
    const tbody = getProductFilesElement('product-files-tbody');
    if (!tbody) return;
    tbody.innerHTML = '<tr><td colspan="7" class="text-center">불러오는 중...</td></tr>';
}

async function loadAvailableFileAssets() {
    const select = getProductFilesElement('product-file-select');
    if (!select) {
        availableFileAssets = [];
        return;
    }

    if (!state.token) {
        availableFileAssets = [];
        populateProductFileSelect();
        return;
    }

    select.innerHTML = '<option value="">파일 목록을 불러오는 중...</option>';

    try {
        const response = await fetch(`/api/admin/files?limit=${FILE_FETCH_LIMIT}`, {
            headers: {
                'Authorization': `Bearer ${state.token}`
            }
        });

        const result = await response.json();

        if (!response.ok || result.status !== 'success') {
            throw new Error(result.message || result.error || '파일 목록을 불러오는데 실패했습니다.');
        }

        availableFileAssets = Array.isArray(result.data) ? result.data : [];
    } catch (error) {
        console.error('Failed to load file assets:', error);
        availableFileAssets = [];
        await showAlert(error.message || '파일 목록을 불러오는데 실패했습니다.', '오류');
    } finally {
        populateProductFileSelect();
    }
}

async function loadProductFileMappings(productId) {
    const targetId = String(productId || '');
    if (!targetId || !state.token) {
        currentProductFileMappings = [];
        renderProductFilesTable();
        return;
    }

    renderProductFilesLoading();

    try {
        const response = await fetch(`/api/admin/product-files?product_id=${encodeURIComponent(targetId)}`, {
            headers: {
                'Authorization': `Bearer ${state.token}`
            }
        });

        const result = await response.json();

        if (!response.ok || result.status !== 'success') {
            throw new Error(result.message || result.error || '제품 파일을 불러오는데 실패했습니다.');
        }

        if (targetId !== currentProductFilesProductId) {
            return;
        }

        currentProductFileMappings = Array.isArray(result.data) ? result.data : [];
    } catch (error) {
        console.error('Failed to load product files:', error);
        currentProductFileMappings = [];
        await showAlert(error.message || '제품 파일을 불러오는데 실패했습니다.', '오류');
    } finally {
        ensureMappingsFilesExistInSelect();
        renderProductFilesTable();
        renderProductFilesSummary(); // 통계도 함께 갱신
    }
}

function renderProductFilesTable() {
    const tbody = getProductFilesElement('product-files-tbody');
    if (!tbody) return;

    tbody.innerHTML = '';

    const mappings = getFilteredProductFileMappings().slice();

    if (!mappings.length) {
        const message = currentProductFilesFilter.trim()
            ? '검색 조건에 맞는 연결이 없습니다.'
            : '연결된 파일이 없습니다.';
        tbody.innerHTML = `<tr><td colspan="7" class="text-center">${message}</td></tr>`;
        renderProductFilesSummary();
        return;
    }

    mappings
        .sort((a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.label.localeCompare(b.label))
        .forEach((mapping) => {
            const row = document.createElement('tr');
            row.className = mapping.is_active ? 'product-file-row is-active' : 'product-file-row is-inactive';
            const file = mapping.file || {};
            const fileName = file.original_name ? escapeHtml(file.original_name) : escapeHtml(mapping.file_id);
            const description = mapping.description ? escapeHtml(mapping.description) : '-';
            const mimeType = file.mime_type && file.mime_type !== 'application/octet-stream'
                ? escapeHtml(file.mime_type)
                : '';
            const downloadLink = file.id
                ? `<button type="button" class="btn btn-sm btn-primary product-file-download" data-action="product-file-download" data-file-id="${escapeHtml(String(file.id))}" data-file-name="${escapeHtml(file.original_name || mapping.file_id)}">다운로드</button>`
                : '<span class="muted-text">다운로드 없음</span>';
            const externalLink = mapping.delivery_url
                ? `<a href="${escapeHtml(mapping.delivery_url)}" target="_blank" rel="noopener noreferrer" class="btn btn-sm btn-secondary product-file-external">외부 다운로드</a>`
                : '<span class="muted-text">외부 링크 없음</span>';
            row.innerHTML = `
                <td>${escapeHtml(mapping.label || '')}</td>
                <td>
                    <div class="product-file-name">${fileName}</div>
                    ${mimeType ? `<div class="product-file-meta">${mimeType}</div>` : ''}
                </td>
                <td>${description}</td>
                <td>${mapping.sort_order ?? 0}</td>
                <td>${downloadLink}</td>
                <td>${externalLink}</td>
                <td>
                    <div class="actions-cell">
                        <button class="btn btn-sm btn-danger" onclick="deleteProductFileMapping('${mapping.id}')">삭제</button>
                    </div>
                </td>
            `;

            tbody.appendChild(row);
        });

        renderProductFilesSummary();
}

function resetProductFileForm({ preserveSelection = false } = {}) {
    const form = getProductFilesElement('product-file-form');
    if (!form) return;

    form.dataset.mode = 'create';

    const idInput = getProductFilesElement('product-file-id');
    if (idInput) idInput.value = '';

    const fileSelect = getProductFilesElement('product-file-select');
    if (fileSelect) {
        fileSelect.disabled = false;
        if (!preserveSelection) {
            fileSelect.value = '';
        }
    }

    const labelInput = getProductFilesElement('product-file-label');
    if (labelInput) labelInput.value = '';

    const descriptionInput = getProductFilesElement('product-file-description');
    if (descriptionInput) descriptionInput.value = '';

    const sortInput = getProductFilesElement('product-file-sort');
    if (sortInput) sortInput.value = '0';

    const deliveryInput = getProductFilesElement('product-file-delivery-url');
    if (deliveryInput) deliveryInput.value = '';

    const submitBtn = getProductFilesElement('product-file-submit');
    if (submitBtn) submitBtn.textContent = '연결 추가';

    const titleEl = getProductFilesElement('product-file-form-title');
    if (titleEl) titleEl.textContent = '새 파일 연결';

    const helperEl = getProductFilesElement('product-file-file-helper');
    if (helperEl) helperEl.textContent = '제품에 연결할 파일을 선택하세요.';

    const modeChip = getProductFilesElement('product-file-form-mode');
    if (modeChip) {
        modeChip.textContent = '새 연결';
        modeChip.classList.remove('is-edit');
        modeChip.classList.add('is-create');
    }

    const clearBtn = getProductFilesElement('product-file-clear');
    if (clearBtn) clearBtn.disabled = true;

    const searchInput = getProductFilesElement('product-file-search');
    if (searchInput && !preserveSelection) {
        searchInput.value = '';
    }
}

function populateProductFileForm(mapping) {
    if (!mapping) return;

    const form = getProductFilesElement('product-file-form');
    if (form) {
        form.dataset.mode = 'edit';
    }

    const idInput = getProductFilesElement('product-file-id');
    if (idInput) idInput.value = mapping.id;

    const select = getProductFilesElement('product-file-select');
    ensureOptionForFile(mapping.file_id, mapping.file, select);
    if (select) {
        select.value = mapping.file_id;
        select.disabled = true;
    }

    const labelInput = getProductFilesElement('product-file-label');
    if (labelInput) labelInput.value = mapping.label || '';

    const descriptionInput = getProductFilesElement('product-file-description');
    if (descriptionInput) descriptionInput.value = mapping.description || '';

    const sortInput = getProductFilesElement('product-file-sort');
    if (sortInput) sortInput.value = (mapping.sort_order ?? 0).toString();

    const deliveryInput = getProductFilesElement('product-file-delivery-url');
    if (deliveryInput) deliveryInput.value = mapping.delivery_url || '';

    const submitBtn = getProductFilesElement('product-file-submit');
    if (submitBtn) submitBtn.textContent = '연결 수정';

    const titleEl = getProductFilesElement('product-file-form-title');
    if (titleEl) titleEl.textContent = '파일 연결 수정';

    const helperEl = getProductFilesElement('product-file-file-helper');
    if (helperEl) helperEl.textContent = '필요한 정보만 수정하고 저장하세요. 다른 파일로 교체하려면 새 연결을 눌러주세요.';

    const modeChip = getProductFilesElement('product-file-form-mode');
    if (modeChip) {
        modeChip.textContent = '연결 수정';
        modeChip.classList.remove('is-create');
        modeChip.classList.add('is-edit');
    }
}

async function manageProductFiles(productId, productName) {
    if (!productId) return;

    const targetId = String(productId);

    if (currentProductFilesProductId === targetId && currentProductFilesRoot) {
        collapseProductFilesPanel();
        return;
    }

    const tableBody = document.getElementById('products-table');
    if (!tableBody) return;

    collapseProductFilesPanel();

    const hostRow = Array.from(tableBody.querySelectorAll('tr')).find((tr) => tr.dataset.productId === targetId);
    if (!hostRow) return;

    const detailRow = document.createElement('tr');
    detailRow.className = 'product-files-detail-row';
    detailRow.dataset.productFilesRow = targetId;

    const detailCell = document.createElement('td');
    detailCell.colSpan = hostRow.children.length || 5;
    detailCell.innerHTML = buildProductFilesPanelMarkup(productName);
    detailRow.appendChild(detailCell);

    hostRow.insertAdjacentElement('afterend', detailRow);

    currentProductFilesRoot = detailCell.querySelector('[data-role="product-files-root"]');
    activeProductFilesRow = detailRow;
    currentProductFilesProductId = targetId;
    currentProductFilesProductName = productName || '';
    currentProductFilesFilter = '';
    currentProductFileMappings = [];

    attachProductFilesPanelEvents();
    resetProductFileForm();
    renderProductFilesSummary();
    renderProductFilesLoading();
    updateManageButtonState(targetId, true);

    await loadAvailableFileAssets();
    await loadProductFileMappings(targetId);
}

async function handleProductFileSubmit(event) {
    event.preventDefault();

    if (!currentProductFilesProductId) {
        await showAlert('먼저 제품을 선택해주세요.', '안내');
        return;
    }

    const id = getProductFilesElement('product-file-id')?.value.trim();
    const fileSelect = getProductFilesElement('product-file-select');
    const fileId = fileSelect?.value.trim();
    const label = getProductFilesElement('product-file-label')?.value.trim();
    const description = getProductFilesElement('product-file-description')?.value.trim() || '';
    const sortValue = Number(getProductFilesElement('product-file-sort')?.value ?? 0);
    const sortOrder = Number.isNaN(sortValue) ? 0 : sortValue;
    const deliveryUrl = getProductFilesElement('product-file-delivery-url')?.value.trim() || '';
    const parseIsActiveValue = (value) => {
        if (typeof value === 'boolean') return value;
        if (typeof value === 'number') return value === 1;
        if (typeof value === 'string') {
            const lowered = value.toLowerCase();
            return lowered === '1' || lowered === 'true';
        }
        return false;
    };
    let isActive = true;
    if (id) {
        const existing = currentProductFileMappings.find((item) => String(item.id) === id);
        if (existing && typeof existing.is_active !== 'undefined') {
            isActive = parseIsActiveValue(existing.is_active);
        }
    }

    if (!fileId) {
        await showAlert('연결할 파일을 선택하세요.', '필수 입력');
        return;
    }

    if (!label) {
        await showAlert('표시명을 입력하세요.', '필수 입력');
        return;
    }

    const submitBtn = getProductFilesElement('product-file-submit');
    const originalText = submitBtn?.textContent;

    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.textContent = '저장 중...';
    }

    try {
        const headers = {
            'Authorization': `Bearer ${state.token}`,
            'Content-Type': 'application/json'
        };

        let response;

        if (id) {
            const payload = {
                id,
                label,
                description,
                sort_order: sortOrder,
                delivery_url: deliveryUrl,
                is_active: isActive
            };

            response = await fetch('/api/admin/product-files', {
                method: 'PUT',
                headers,
                body: JSON.stringify(payload)
            });
        } else {
            const payload = {
                product_id: currentProductFilesProductId,
                file_id: fileId,
                label,
                description,
                sort_order: sortOrder,
                delivery_url: deliveryUrl,
                is_active: isActive
            };

            response = await fetch('/api/admin/product-files', {
                method: 'POST',
                headers,
                body: JSON.stringify(payload)
            });
        }

        const result = await response.json();

        if (!response.ok || result.status !== 'success') {
            throw new Error(result.message || result.error || '요청을 처리하는 중 오류가 발생했습니다.');
        }

        await loadProductFileMappings(currentProductFilesProductId);
        populateProductFileSelect();

        await showAlert(id ? '파일 연결이 수정되었습니다.' : '파일이 제품에 연결되었습니다.', '완료');
        resetProductFileForm({ preserveSelection: !id });
    } catch (error) {
        console.error('Failed to save product file mapping:', error);
        await showAlert(error.message || '요청을 처리하는 중 오류가 발생했습니다.', '오류');
    } finally {
        if (submitBtn) {
            submitBtn.disabled = false;
            if (originalText) submitBtn.textContent = originalText;
        }
    }
}

async function deleteProductFileMapping(mappingId) {
    const mapping = currentProductFileMappings.find((item) => item.id === mappingId);
    if (!mapping) return;

    const confirmed = await showConfirm(`"${mapping.label}" 연결을 삭제하시겠습니까?`, '연결 삭제');
    if (!confirmed) return;

    try {
        const response = await fetch(`/api/admin/product-files?id=${encodeURIComponent(mappingId)}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${state.token}`
            }
        });

        const result = await response.json();

        if (!response.ok || result.status !== 'success') {
            throw new Error(result.message || result.error || '연결을 삭제하는 중 오류가 발생했습니다.');
        }

        await loadProductFileMappings(currentProductFilesProductId);
        populateProductFileSelect();
        await showAlert('연결이 삭제되었습니다.', '완료');
        resetProductFileForm();
    } catch (error) {
        console.error('Failed to delete product file mapping:', error);
        await showAlert(error.message || '연결을 삭제하는 중 오류가 발생했습니다.', '오류');
    }
}

function editProductFileMapping(mappingId) {
    const mapping = currentProductFileMappings.find((item) => item.id === mappingId);
    if (!mapping) return;
    populateProductFileForm(mapping);
}

function handleProductFilesSearch(event) {
    currentProductFilesFilter = event.target.value || '';
    renderProductFilesTable();
    const clearBtn = getProductFilesElement('product-file-clear');
    if (clearBtn) {
        clearBtn.disabled = currentProductFilesFilter.trim().length === 0;
    }
}

function initProductsPage() {
    loadProducts();

    // 생성 버튼 클릭 이벤트
    const createBtn = document.querySelector('#create-product-btn');
    if (createBtn) {
        createBtn.addEventListener('click', (e) => {
            e.preventDefault();
            showProductModal();
        });
    }

    // NOTE: product-form 제출 이벤트는 main.js의 setupEventListeners()에서 등록됨 (중복 방지)
}

// 전역으로 노출 (main.js에서 window.loadProducts 호출용)
window.loadProducts = loadProducts;
window.showProductModal = showProductModal;
window.closeProductModal = closeProductModal;
window.editProduct = editProduct;
window.deleteProduct = deleteProduct;
window.openProductModal = showProductModal; // legacy alias for main.js
window.handleCreateProduct = handleCreateProduct;
window.manageProductFiles = manageProductFiles;
window.handleProductFileSubmit = handleProductFileSubmit;
window.deleteProductFileMapping = deleteProductFileMapping;
window.editProductFileMapping = editProductFileMapping;
window.resetProductFileForm = resetProductFileForm;

export { loadProducts, showProductModal, initProductsPage, handleCreateProduct };
window.deleteProductConfirm = deleteProductConfirm;
