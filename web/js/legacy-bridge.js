// This file only imports the existing monolithic scripts to keep behavior working
// while we migrate features into modules.
import '../app.js';
import '../products.js'; // 제품 관리 (openProductModal, handleCreateProduct 등)
import './pages/products.js';
