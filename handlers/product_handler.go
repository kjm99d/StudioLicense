package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/services"
	"studiolicense/utils"
)

// ProductHandler는 제품 관련 HTTP 요청을 처리한다.
type ProductHandler struct {
	service       services.ProductService
	scopeResolver services.ResourceScopeResolver
}

// NewProductHandler는 제품 핸들러를 생성한다.
func NewProductHandler(service services.ProductService, resolver services.ResourceScopeResolver) *ProductHandler {
	return &ProductHandler{
		service:       service,
		scopeResolver: resolver,
	}
}

// Create 제품 생성
// @Summary 제품 생성
// @Description 새로운 제품을 생성합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateProductRequest true "제품 정보"
// @Success 201 {object} models.APIResponse{data=models.Product} "생성 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 409 {object} models.APIResponse "중복 제품명"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	creatorID, _ := r.Context().Value("admin_id").(string)
	if strings.TrimSpace(creatorID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Missing creator context", nil))
		return
	}

	product, err := h.service.Create(r.Context(), req, creatorID)
	if err != nil {
		if errors.Is(err, services.ErrProductNameConflict) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ErrorResponse("이미 존재하는 제품명입니다", nil))
			return
		}
		logger.WithFields(map[string]interface{}{"error": err.Error(), "name": req.Name}).Error("Failed to create product")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to create product", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"product_id": product.ID,
		"name":       product.Name,
	}).Info("Product created")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("Product created successfully", product))

	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionCreateProduct, "Product created: "+product.ID)
	}
}

// List 제품 목록 조회
// @Summary 제품 목록 조회
// @Description 모든 제품 목록을 조회합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "상태 필터 (active, inactive)"
// @Success 200 {object} models.APIResponse{data=[]models.Product} "조회 성공"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	role := getRole(r)
	adminID := getAdminID(r)
	scope, isSuper, err := h.scopeResolver.Resolve(r.Context(), role, adminID, models.ResourceTypeProducts)
	if err != nil {
		logger.Error("Failed to evaluate product permissions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate product permissions", err))
		return
	}

	products, err := h.service.List(r.Context(), services.ProductFilter{
		Status:  status,
		Scope:   scope,
		IsSuper: isSuper,
		AdminID: adminID,
	})
	if err != nil {
		logger.Error("Failed to query products: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query products", err))
		return
	}

	logger.Info("Retrieved %d products", len(products))
	json.NewEncoder(w).Encode(models.SuccessResponse("Products retrieved", products))
}

// Get 제품 상세 조회
// @Summary 제품 상세 조회
// @Description 특정 제품의 상세 정보를 조회합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "제품 ID"
// @Success 200 {object} models.APIResponse{data=models.Product} "조회 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 403 {object} models.APIResponse "권한 없음"
// @Failure 404 {object} models.APIResponse "제품 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products/ [get]
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	role := getRole(r)
	adminID := getAdminID(r)
	scope, isSuper, err := h.scopeResolver.Resolve(r.Context(), role, adminID, models.ResourceTypeProducts)
	if err != nil {
		logger.Error("Failed to evaluate product permissions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate product permissions", err))
		return
	}

	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeNone) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: product access denied", nil))
		return
	}

	product, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Product not found", nil))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to retrieve product", err))
		return
	}

	if !isSuper && !utils.CanAccessResource(scope, product.ID, product.CreatedBy, adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: product access denied", nil))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Product retrieved", product))
}

// Update 제품 수정
// @Summary 제품 수정
// @Description 제품 정보를 수정합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "제품 ID"
// @Param request body models.UpdateProductRequest true "수정할 정보"
// @Success 200 {object} models.APIResponse "수정 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "제품 없음"
// @Failure 409 {object} models.APIResponse "중복 제품명"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products/ [put]
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	var req models.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	err := h.service.Update(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProductNameConflict):
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ErrorResponse("이미 존재하는 제품명입니다", nil))
			return
		case errors.Is(err, services.ErrProductNotFound):
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Product not found", nil))
			return
		default:
			logger.WithFields(map[string]interface{}{
				"error":      err.Error(),
				"product_id": id,
			}).Error("Failed to update product")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update product", err))
			return
		}
	}

	logger.WithFields(map[string]interface{}{"product_id": id}).Info("Product updated")
	json.NewEncoder(w).Encode(models.SuccessResponse("Product updated successfully", nil))

	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionUpdateProduct, "Product updated: "+id)
	}
}

// Delete 제품 삭제
// @Summary 제품 삭제
// @Description 제품을 삭제합니다. 연결된 라이선스가 있으면 삭제가 제한됩니다(RESTRICT).
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "제품 ID"
// @Success 200 {object} models.APIResponse "삭제 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "제품 없음"
// @Failure 409 {object} models.APIResponse "연결된 라이선스로 인해 삭제 불가"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products/ [delete]
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if strings.TrimSpace(id) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	err := h.service.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProductLinkedLicenses):
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ErrorResponse("Cannot delete product: linked licenses exist", nil))
			return
		case errors.Is(err, services.ErrProductNotFound):
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Product not found", nil))
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete product", err))
			return
		}
	}

	logger.WithFields(map[string]interface{}{"product_id": id}).Info("Product deleted")
	json.NewEncoder(w).Encode(models.SuccessResponse("Product deleted successfully", nil))

	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionDeleteProduct, "Product deleted: "+id)
	}
}

func getRole(r *http.Request) string {
	role, _ := r.Context().Value("role").(string)
	return role
}

func getAdminID(r *http.Request) string {
	adminID, _ := r.Context().Value("admin_id").(string)
	return adminID
}
