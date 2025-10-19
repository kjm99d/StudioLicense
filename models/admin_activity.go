package models

// AdminActivityLog 관리자 활동 로그
type AdminActivityLog struct {
	ID        int64  `json:"id" db:"id"`
	AdminID   string `json:"admin_id" db:"admin_id"`
	Username  string `json:"username" db:"username"`
	Action    string `json:"action" db:"action"`
	Details   string `json:"details" db:"details"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

// 관리자 활동 액션 상수
const (
	AdminActionLogin            = "login"
	AdminActionChangePassword   = "change_password"
	AdminActionCreateProduct    = "create_product"
	AdminActionUpdateProduct    = "update_product"
	AdminActionDeleteProduct    = "delete_product"
	AdminActionCreateLicense    = "create_license"
	AdminActionUpdateLicense    = "update_license"
	AdminActionDeleteLicense    = "delete_license"
	AdminActionDeactivateDev    = "deactivate_device"
	AdminActionDeactivateDevice = "deactivate_device"
	AdminActionReactivateDev    = "reactivate_device"
	AdminActionCleanupDevices   = "cleanup_devices"
	AdminActionCreatePolicy     = "create_policy"
	AdminActionUpdatePolicy     = "update_policy"
	AdminActionDeletePolicy     = "delete_policy"
	AdminActionCreateAdmin      = "create_admin"
	AdminActionResetPassword    = "reset_password"
	AdminActionDeleteAdmin      = "delete_admin"
)
