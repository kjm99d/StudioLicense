
// ==================== 제품 관리 ====================

// 제품 목록 로드
window.loadProducts = async function loadProducts() {
    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/products`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) throw new Error('Failed to load products');

        const result = await response.json();
        displayProducts(result.data || []);
    } catch (error) {
        console.error('Error loading products:', error);
        document.getElementById('products-table').innerHTML = `
            <tr><td colspan="6" class="text-center error">제품 목록을 불러오는데 실패했습니다</td></tr>
        `;
    }
}

// 제품 목록 표시
function displayProducts(products) {
    const tbody = document.getElementById('products-table');
    
    if (!products || products.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center">등록된 제품이 없습니다</td></tr>';
        return;
    }

    tbody.innerHTML = products.map(product => `
        <tr>
            <td><strong>${product.name}</strong></td>
            <td>${product.description || '-'}</td>
            <td>${renderStatusBadge(product.status)}</td>
            <td>${formatDate(product.created_at)}</td>
            <td>
                <button class="btn btn-sm btn-secondary" onclick="editProduct('${product.id}')">수정</button>
                <button class="btn btn-sm btn-danger" onclick="deleteProduct('${product.id}')">삭제</button>
            </td>
        </tr>
    `).join('');
}

// 제품 생성 모달 열기
window.openProductModal = function openProductModal(productId = null) {
    const modal = document.getElementById('product-modal');
    const form = document.getElementById('product-form');
    const title = document.getElementById('product-modal-title');
    
    form.reset();
    
    if (productId) {
        title.textContent = '제품 수정';
        // TODO: 제품 정보 로드 및 폼 채우기
    } else {
        title.textContent = '새 제품 생성';
    }
    
    modal.classList.add('active');
}

// 제품 생성 처리
window.handleCreateProduct = async function handleCreateProduct(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const data = {
        name: formData.get('name'),
        description: formData.get('description')
    };

    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/products`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            const error = await response.json();
            // 409 Conflict: 제품명 중복
            if (response.status === 409) {
                const modal = document.getElementById('product-modal');
                if (modal) modal.classList.remove('active');
                e.target.reset();
                
                setTimeout(() => {
                    showAlert('이미 존재하는 제품명입니다. 다른 이름을 사용해주세요.', '제품 생성 실패');
                }, 300);
                return;
            }
            throw new Error(error.message || 'Failed to create product');
        }

        const result = await response.json();
        const modal = document.getElementById('product-modal');
        if (modal) modal.classList.remove('active');
        e.target.reset();
        
        setTimeout(async () => {
            await showAlert('제품이 생성되었습니다!', '제품 생성 완료');
            loadProducts();
        }, 300);
    } catch (error) {
        console.error('Error creating product:', error);
        const modal = document.getElementById('product-modal');
        if (modal) modal.classList.remove('active');
        e.target.reset();
        
        setTimeout(() => {
            showAlert('제품 생성 실패: ' + error.message, '제품 생성 실패');
        }, 300);
    }
}

// 제품 수정
window.editProduct = async function editProduct(productId) {
    // TODO: 구현
    await showAlert('제품 수정 기능은 곧 추가됩니다.', '제품 수정');
}

// 제품 삭제
window.deleteProduct = async function deleteProduct(productId) {
    const ok = await showConfirm('정말 이 제품을 삭제하시겠습니까?\n\n주의: 이 제품을 참조하는 라이선스가 존재하면 삭제할 수 없습니다.', '제품 삭제');
    if (!ok) return;

    try {
        const response = await apiFetch(`${API_BASE_URL}/api/admin/products/?id=${productId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) {
            const body = await response.json().catch(() => ({ message: '삭제 실패' }));
            if (response.status === 409) {
                await showAlert('삭제할 수 없습니다: 이 제품을 참조하는 라이선스가 존재합니다. 먼저 관련 라이선스를 처리하세요.', '제품 삭제');
                return;
            }
            throw new Error(body.message || 'Failed to delete product');
        }

        await showAlert('제품이 삭제되었습니다.', '제품 삭제');
        loadProducts();
    } catch (error) {
        console.error('Error deleting product:', error);
        await showAlert('제품 삭제 실패: ' + error.message, '제품 삭제');
    }
}
