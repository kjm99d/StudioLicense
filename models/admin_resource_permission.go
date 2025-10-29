package models

import (
	"sort"
	"strings"
)

// Resource type keys used for resource-based permissions.
const (
	ResourceTypeLicenses = "licenses"
	ResourceTypePolicies = "policies"
	ResourceTypeProducts = "products"
)

// AdminResourceTypes lists all supported resource types.
var AdminResourceTypes = []string{
	ResourceTypeLicenses,
	ResourceTypePolicies,
	ResourceTypeProducts,
}

// Resource mode keys describing how access should be granted.
const (
	ResourceModeAll    = "all"
	ResourceModeNone   = "none"
	ResourceModeOwn    = "own"
	ResourceModeCustom = "custom"
)

var validResourceModes = map[string]struct{}{
	ResourceModeAll:    {},
	ResourceModeNone:   {},
	ResourceModeOwn:    {},
	ResourceModeCustom: {},
}

// AdminResourcePermissionConfig represents access control for a resource type.
type AdminResourcePermissionConfig struct {
	Mode        string   `json:"mode"`
	SelectedIDs []string `json:"selected_ids"`
}

// IsValidResourceType returns true when the resource type is supported.
func IsValidResourceType(resourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case ResourceTypeLicenses, ResourceTypePolicies, ResourceTypeProducts:
		return true
	default:
		return false
	}
}

// IsValidResourceMode returns true for a known access mode.
func IsValidResourceMode(mode string) bool {
	_, ok := validResourceModes[strings.ToLower(strings.TrimSpace(mode))]
	return ok
}

// NormalizeAdminResourcePermissions sanitizes incoming payloads and applies defaults.
func NormalizeAdminResourcePermissions(input map[string]AdminResourcePermissionConfig) map[string]AdminResourcePermissionConfig {
	result := make(map[string]AdminResourcePermissionConfig, len(AdminResourceTypes))
	for _, resourceType := range AdminResourceTypes {
		cfg := AdminResourcePermissionConfig{}
		if input != nil {
			if provided, ok := input[resourceType]; ok {
				cfg = provided
			}
		}

		mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
		if !IsValidResourceMode(mode) {
			mode = ResourceModeAll
		}

		var selected []string
		if mode == ResourceModeCustom {
			dedup := make(map[string]struct{})
			for _, id := range cfg.SelectedIDs {
				id = strings.TrimSpace(id)
				if id == "" {
					continue
				}
				dedup[id] = struct{}{}
			}
			if len(dedup) > 0 {
				selected = make([]string, 0, len(dedup))
				for id := range dedup {
					selected = append(selected, id)
				}
				sort.Strings(selected)
			}
		}

		result[resourceType] = AdminResourcePermissionConfig{
			Mode:        mode,
			SelectedIDs: selected,
		}
	}
	return result
}

// DefaultAdminResourcePermissions returns the default (all-allowed) configuration.
func DefaultAdminResourcePermissions() map[string]AdminResourcePermissionConfig {
	return NormalizeAdminResourcePermissions(nil)
}
