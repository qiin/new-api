package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

// TestCalculateUserPermissionsSidebarPersonalization verifies the site-wide
// SidebarPersonalizationEnabled toggle hides the sidebar customization card
// for admin and normal users while leaving root's own hard-coded exclusion
// unaffected, and restores per-role behavior once re-enabled.
func TestCalculateUserPermissionsSidebarPersonalization(t *testing.T) {
	original := common.SidebarPersonalizationEnabled
	t.Cleanup(func() {
		common.SidebarPersonalizationEnabled = original
	})

	testCases := []struct {
		name                string
		role                int
		personalizationOn   bool
		expectSidebarAccess bool
	}{
		{
			name:                "root is always excluded regardless of the toggle",
			role:                common.RoleRootUser,
			personalizationOn:   true,
			expectSidebarAccess: false,
		},
		{
			name:                "admin can configure sidebar when enabled",
			role:                common.RoleAdminUser,
			personalizationOn:   true,
			expectSidebarAccess: true,
		},
		{
			name:                "normal user can configure sidebar when enabled",
			role:                common.RoleCommonUser,
			personalizationOn:   true,
			expectSidebarAccess: true,
		},
		{
			name:                "admin loses sidebar settings when the toggle is off",
			role:                common.RoleAdminUser,
			personalizationOn:   false,
			expectSidebarAccess: false,
		},
		{
			name:                "normal user loses sidebar settings when the toggle is off",
			role:                common.RoleCommonUser,
			personalizationOn:   false,
			expectSidebarAccess: false,
		},
		{
			name:                "root stays excluded when the toggle is off too",
			role:                common.RoleRootUser,
			personalizationOn:   false,
			expectSidebarAccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			common.SidebarPersonalizationEnabled = tc.personalizationOn

			permissions := calculateUserPermissions(tc.role)

			require.Equal(t, tc.expectSidebarAccess, permissions["sidebar_settings"])
		})
	}
}
