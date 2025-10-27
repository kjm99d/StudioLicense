package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// ActivateLicense ?쇱씠?좎뒪 ?쒖꽦??(理쒖큹 ?깅줉)
// @Summary ?쇱씠?좎뒪 ?쒖꽦??
// @Description ?쇱씠?좎뒪 ?ㅻ? ?ъ슜?섏뿬 ?붾컮?댁뒪瑜??쒖꽦?뷀빀?덈떎
// @Tags ?대씪?댁뼵??- ?쇱씠?좎뒪
// @Accept json
// @Produce json
// @Param request body models.ActivateRequest true "?쒖꽦???뺣낫"
// @Success 201 {object} models.APIResponse "?쒖꽦???깃났"
// @Success 200 {object} models.APIResponse "?대? ?쒖꽦?붾맖"
// @Failure 400 {object} models.APIResponse "?섎せ???붿껌"
// @Failure 403 {object} models.APIResponse "?쇱씠?좎뒪 鍮꾪솢??留뚮즺/湲곌린??珥덇낵"
// @Failure 404 {object} models.APIResponse "?쇱씠?좎뒪 ?놁쓬"
// @Failure 500 {object} models.APIResponse "?쒕쾭 ?먮윭"
// @Router /api/license/activate [post]
func ActivateLicense(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req models.ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("Invalid activate request")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
		"device_name": req.DeviceInfo.Hostname,
	}).Info("License activation attempt")

	// ?쇱씠?좎뒪 議고쉶
	var license models.License
	var productID sql.NullString
	var policyID sql.NullString
	query := `SELECT id, license_key, product_id, policy_id, product_name, customer_name, max_devices, 
		expires_at, status FROM licenses WHERE license_key = ?`

	err := database.DB.QueryRow(query, req.LicenseKey).Scan(
		&license.ID,
		&license.LicenseKey,
		&productID,
		&policyID,
		&license.ProductName,
		&license.CustomerName,
		&license.MaxDevices,
		&license.ExpiresAt,
		&license.Status,
	)

	if err == sql.ErrNoRows {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
		}).Warn("License not found")

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("License not found", nil))
		return
	}

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to query license")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query license", err))
		return
	}

	if productID.Valid {
		license.ProductID = new(string)
		*license.ProductID = productID.String
	}

	if policyID.Valid {
		license.PolicyID = new(string)
		*license.PolicyID = policyID.String
	}

	// ?쇱씠?좎뒪 ?좏슚??寃??
	if license.Status != models.LicenseStatusActive {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"status":      license.Status,
		}).Warn("License is not active")

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License is not active", nil))
		return
	}

	if license.IsExpired() {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"expires_at":  license.ExpiresAt,
		}).Warn("License has expired")

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License has expired", nil))
		return
	}

	// ?붾컮?댁뒪 ?묎굅?꾨┛???앹꽦
	fingerprint := utils.GenerateDeviceFingerprint(
		req.DeviceInfo.ClientID,
		req.DeviceInfo.CPUID,
		req.DeviceInfo.MotherboardSN,
		req.DeviceInfo.MACAddress,
		req.DeviceInfo.DiskSerial,
		req.DeviceInfo.MachineID,
	)

	// ?대? ?쒖꽦?붾맂 ?붾컮?댁뒪?몄? ?뺤씤
	var existingID string
	checkQuery := "SELECT id FROM device_activations WHERE license_id = ? AND device_fingerprint = ? AND status = ?"
	err = database.DB.QueryRow(checkQuery, license.ID, fingerprint, models.DeviceStatusActive).Scan(&existingID)

	if err == nil {
		// ?대? ?쒖꽦?붾맖
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"device_id":   existingID,
		}).Info("Device already activated")

		json.NewEncoder(w).Encode(models.SuccessResponse("Device already activated", map[string]interface{}{
			"license_key":   license.LicenseKey,
			"device_id":     existingID,
			"expires_at":    license.ExpiresAt,
			"product_id":    stringValue(license.ProductID),
			"policies":      loadPoliciesForLicense(license.PolicyID),
			"product_files": loadProductFilesForProduct(license.ProductID),
		}))
		return
	}

	if err != nil && err != sql.ErrNoRows {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to verify device activation")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify device activation", err))
		return
	}

	// ?꾩옱 ?쒖꽦?붾맂 ?붾컮?댁뒪 ???뺤씤
	var activeCount int
	countQuery := "SELECT COUNT(*) FROM device_activations WHERE license_id = ? AND status = ?"
	err = database.DB.QueryRow(countQuery, license.ID, models.DeviceStatusActive).Scan(&activeCount)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to count active devices")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to count active devices", err))
		return
	}

	if activeCount >= license.MaxDevices {
		logger.WithFields(map[string]interface{}{
			"request_id":   requestID,
			"license_key":  req.LicenseKey,
			"active_count": activeCount,
			"max_devices":  license.MaxDevices,
		}).Warn("Maximum device limit reached")

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Maximum device limit reached", nil))
		return
	}

	// ?붾컮?댁뒪 ?뺣낫 JSON 吏곷젹??
	deviceInfoJSON, _ := json.Marshal(req.DeviceInfo)

	// ?붾컮?댁뒪 ?쒖꽦??
	deviceID, _ := utils.GenerateID("dev")
	insertQuery := `INSERT INTO device_activations 
		(id, license_id, device_fingerprint, device_info, device_name, status, activated_at, last_validated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = database.DB.Exec(insertQuery,
		deviceID, license.ID, fingerprint, string(deviceInfoJSON),
		req.DeviceInfo.Hostname, models.DeviceStatusActive, now, now,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to activate device")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to activate device", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
		"device_id":   deviceID,
		"device_name": req.DeviceInfo.Hostname,
		"customer":    license.CustomerName,
	}).Info("License activated successfully")

	// ?쒕룞 濡쒓렇 湲곕줉
	utils.LogDeviceActivity(deviceID, license.ID, models.DeviceActionActivated, "Device activated")

	policies := loadPoliciesForLicense(license.PolicyID)
	productFiles := loadProductFilesForProduct(license.ProductID)
	productIDValue := stringValue(license.ProductID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("License activated successfully", map[string]interface{}{
		"license_key":   license.LicenseKey,
		"device_id":     deviceID,
		"product_id":    productIDValue,
		"product_name":  license.ProductName,
		"expires_at":    license.ExpiresAt,
		"policies":      policies,
		"product_files": productFiles,
	}))
}

// ValidateLicense ?쇱씠?좎뒪 寃利?(???ㅽ뻾 ??
// @Summary ?쇱씠?좎뒪 寃利?
// @Description ???ㅽ뻾 ???쇱씠?좎뒪瑜?寃利앺빀?덈떎
// @Tags ?대씪?댁뼵??- ?쇱씠?좎뒪
// @Accept json
// @Produce json
// @Param request body models.ValidateRequest true "寃利??뺣낫"
// @Success 200 {object} models.APIResponse "寃利??깃났"
// @Failure 400 {object} models.APIResponse "?섎せ???붿껌"
// @Failure 403 {object} models.APIResponse "?쇱씠?좎뒪 鍮꾪솢??留뚮즺 ?먮뒗 ?붾컮?댁뒪 誘몃벑濡?"
// @Failure 404 {object} models.APIResponse "?쇱씠?좎뒪 ?놁쓬"
// @Failure 500 {object} models.APIResponse "?쒕쾭 ?먮윭"
// @Router /api/license/validate [post]
func ValidateLicense(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req models.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
	}).Debug("License validation request")

	// ?쇱씠?좎뒪 議고쉶
	var license models.License
	var productID sql.NullString
	var policyID sql.NullString
	query := `SELECT id, license_key, product_id, policy_id, product_name, expires_at, status FROM licenses WHERE license_key = ?`

	err := database.DB.QueryRow(query, req.LicenseKey).Scan(
		&license.ID,
		&license.LicenseKey,
		&productID,
		&policyID,
		&license.ProductName,
		&license.ExpiresAt,
		&license.Status,
	)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("License not found", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query license", err))
		return
	}

	if productID.Valid {
		license.ProductID = new(string)
		*license.ProductID = productID.String
	}

	if policyID.Valid {
		license.PolicyID = new(string)
		*license.PolicyID = policyID.String
	}

	// ?쇱씠?좎뒪 ?좏슚??寃??
	if license.Status != models.LicenseStatusActive {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License is not active", nil))
		return
	}

	if license.IsExpired() {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License has expired", nil))
		return
	}

	// ?붾컮?댁뒪 ?묎굅?꾨┛???앹꽦
	fingerprint := utils.GenerateDeviceFingerprint(
		req.DeviceInfo.ClientID,
		req.DeviceInfo.CPUID,
		req.DeviceInfo.MotherboardSN,
		req.DeviceInfo.MACAddress,
		req.DeviceInfo.DiskSerial,
		req.DeviceInfo.MachineID,
	)

	// ?붾컮?댁뒪 ?쒖꽦???щ? ?뺤씤
	var deviceID string
	deviceQuery := "SELECT id FROM device_activations WHERE license_id = ? AND device_fingerprint = ? AND status = ?"
	err = database.DB.QueryRow(deviceQuery, license.ID, fingerprint, models.DeviceStatusActive).Scan(&deviceID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device not activated", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify device", err))
		return
	}

	// 留덉?留?寃利??쒓컙 ?낅뜲?댄듃
	updateQuery := "UPDATE device_activations SET last_validated_at = ? WHERE id = ?"
	database.DB.Exec(updateQuery, time.Now().Format("2006-01-02 15:04:05"), deviceID)

	policies := loadPoliciesForLicense(license.PolicyID)
	productFiles := loadProductFilesForProduct(license.ProductID)
	productIDValue := stringValue(license.ProductID)

	json.NewEncoder(w).Encode(models.SuccessResponse("License is valid", map[string]interface{}{
		"license_key":   license.LicenseKey,
		"product_id":    productIDValue,
		"product_name":  license.ProductName,
		"expires_at":    license.ExpiresAt,
		"valid":         true,
		"policies":      policies,
		"product_files": productFiles,
	}))
}

func loadPoliciesForLicense(policyID *string) []models.PolicyResponse {
	if policyID == nil || *policyID == "" {
		return nil
	}

	query := `SELECT id, policy_name, policy_data FROM policies WHERE id = ?`
	rows, err := database.DB.Query(query, *policyID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"policy_id": *policyID,
			"error":     err.Error(),
		}).Error("Failed to load policies for license")
		return nil
	}
	defer rows.Close()

	policies := []models.PolicyResponse{}
	for rows.Next() {
		var (
			p             models.PolicyResponse
			policyDataStr string
		)
		if err := rows.Scan(&p.ID, &p.PolicyName, &policyDataStr); err != nil {
			logger.Warn("Failed to scan policy row: %v", err)
			continue
		}

		if err := json.Unmarshal([]byte(policyDataStr), &p.PolicyData); err != nil {
			logger.Warn("Failed to parse policy JSON: %v", err)
			continue
		}

		policies = append(policies, p)
	}

	return policies
}

func loadProductFilesForProduct(productID *string) []models.ProductFileResponse {
	if productID == nil || *productID == "" {
		return nil
	}

	query := `SELECT pf.id, pf.file_id, pf.label, pf.description, pf.sort_order, pf.delivery_url, pf.updated_at,
		f.mime_type, f.file_size, f.checksum, f.storage_path
		FROM product_files pf
		JOIN files f ON pf.file_id = f.id
		WHERE pf.product_id = ? AND pf.is_active = 1
		ORDER BY pf.sort_order ASC, pf.created_at DESC`

	rows, err := database.DB.Query(query, *productID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"product_id": *productID,
			"error":      err.Error(),
		}).Error("Failed to load product files for product")
		return nil
	}
	defer rows.Close()

	files := []models.ProductFileResponse{}
	for rows.Next() {
		var (
			item        models.ProductFileResponse
			description sql.NullString
			deliveryURL sql.NullString
			checksum    sql.NullString
		)

		if err := rows.Scan(
			&item.ID,
			&item.FileID,
			&item.Label,
			&description,
			&item.SortOrder,
			&deliveryURL,
			&item.UpdatedAt,
			&item.MimeType,
			&item.FileSize,
			&checksum,
			&item.StoragePath,
		); err != nil {
			logger.Warn("Failed to scan product file mapping: %v", err)
			continue
		}

		if description.Valid {
			item.Description = description.String
		}

		if deliveryURL.Valid {
			item.DeliveryURL = deliveryURL.String
		}

		if checksum.Valid {
			item.Checksum = checksum.String
		}

		downloadURL := fmt.Sprintf("/api/admin/files/%s?download=1", item.FileID)
		if item.DeliveryURL != "" {
			item.URL = item.DeliveryURL
		} else {
			item.URL = downloadURL
		}
		item.DownloadURL = downloadURL

		files = append(files, item)
	}

	return files
}

func stringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
