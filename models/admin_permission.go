package models

// Admin permission keys
const (
	PermissionDashboardView = "dashboard.view"

	PermissionLicensesView   = "licenses.view"
	PermissionLicensesManage = "licenses.manage"

	PermissionDevicesView   = "devices.view"
	PermissionDevicesManage = "devices.manage"

	PermissionProductsView   = "products.view"
	PermissionProductsManage = "products.manage"

	PermissionPoliciesView   = "policies.view"
	PermissionPoliciesManage = "policies.manage"

	PermissionFilesView   = "files.view"
	PermissionFilesManage = "files.manage"

	PermissionClientLogsView   = "client_logs.view"
	PermissionClientLogsManage = "client_logs.manage"
)

// AdminPermissionDefinition describes a permission for UI
type AdminPermissionDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// AdminPermissionCatalog enumerates assignable permissions
var AdminPermissionCatalog = []AdminPermissionDefinition{
	{
		Key:         PermissionDashboardView,
		Label:       "대시보드 보기",
		Description: "대시보드 통계와 최근 활동을 볼 수 있습니다.",
		Category:    "대시보드",
	},
	{
		Key:         PermissionLicensesView,
		Label:       "라이선스 조회",
		Description: "라이선스 목록과 상세 정보를 볼 수 있습니다.",
		Category:    "라이선스",
	},
	{
		Key:         PermissionLicensesManage,
		Label:       "라이선스 관리",
		Description: "라이선스를 생성, 수정, 삭제할 수 있습니다.",
		Category:    "라이선스",
	},
	{
		Key:         PermissionDevicesView,
		Label:       "디바이스 조회",
		Description: "디바이스 목록과 활동 로그를 확인할 수 있습니다.",
		Category:    "디바이스",
	},
	{
		Key:         PermissionDevicesManage,
		Label:       "디바이스 관리",
		Description: "디바이스 활성화 상태를 변경하고 정리를 수행할 수 있습니다.",
		Category:    "디바이스",
	},
	{
		Key:         PermissionProductsView,
		Label:       "제품 조회",
		Description: "제품 목록과 상세 정보를 볼 수 있습니다.",
		Category:    "제품",
	},
	{
		Key:         PermissionProductsManage,
		Label:       "제품 관리",
		Description: "제품을 생성하거나 수정할 수 있습니다.",
		Category:    "제품",
	},
	{
		Key:         PermissionPoliciesView,
		Label:       "정책 조회",
		Description: "정책 목록과 상세 정보를 볼 수 있습니다.",
		Category:    "정책",
	},
	{
		Key:         PermissionPoliciesManage,
		Label:       "정책 관리",
		Description: "정책을 생성, 수정, 삭제할 수 있습니다.",
		Category:    "정책",
	},
	{
		Key:         PermissionFilesView,
		Label:       "파일 서버 조회",
		Description: "제품 파일 목록을 조회하고 다운로드할 수 있습니다.",
		Category:    "파일 서버",
	},
	{
		Key:         PermissionFilesManage,
		Label:       "파일 서버 관리",
		Description: "제품 파일을 업로드하거나 삭제할 수 있습니다.",
		Category:    "파일 서버",
	},
	{
		Key:         PermissionClientLogsView,
		Label:       "클라이언트 로그 조회",
		Description: "클라이언트 로그를 검색하고 확인할 수 있습니다.",
		Category:    "클라이언트 로그",
	},
	{
		Key:         PermissionClientLogsManage,
		Label:       "클라이언트 로그 정리",
		Description: "클라이언트 로그를 일괄 삭제할 수 있습니다.",
		Category:    "클라이언트 로그",
	},
}

// IsValidAdminPermission checks whether key is assignable
func IsValidAdminPermission(key string) bool {
	for _, def := range AdminPermissionCatalog {
		if def.Key == key {
			return true
		}
	}
	return false
}

// AllAdminPermissionKeys returns all permission keys
func AllAdminPermissionKeys() []string {
	keys := make([]string, 0, len(AdminPermissionCatalog))
	for _, def := range AdminPermissionCatalog {
		keys = append(keys, def.Key)
	}
	return keys
}
