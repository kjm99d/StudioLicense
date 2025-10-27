// 제품 관리 페이지

import { state } from '../state.js';
import { showAlert, openModal, closeModal } from '../modals.js';
import { renderStatusBadge } from '../ui.js';
import { formatDate } from '../utils.js';

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

    tbody.innerHTML = '';

    if (!products || products.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; padding: 20px;">제품이 없습니다.</td></tr>';
        return;
    }

    products.forEach(product => {
        const row = document.createElement('tr');
        const statusBadge = renderStatusBadge(product.status);
        const createdDate = formatDate(product.created_at);

        row.innerHTML = `
            <td>${product.name}</td>
            <td>${product.description || '-'}</td>
            <td>${statusBadge}</td>
            <td>${createdDate}</td>
            <td style="white-space: nowrap;">
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

// 페이지 초기화
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

export { loadProducts, showProductModal, initProductsPage, handleCreateProduct };
window.deleteProductConfirm = deleteProductConfirm;

