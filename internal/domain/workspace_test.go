package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleValid(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{name: "owner", role: RoleOwner, want: true},
		{name: "staff", role: RoleStaff, want: true},
		{name: "empty", role: "", want: false},
		{name: "unknown", role: "admin", want: false},
		{name: "spaces", role: " owner", want: false},
		{name: "mixed case", role: "Owner", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.role.Valid())
		})
	}
}

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want []string
	}{
		{
			name: "owner",
			role: RoleOwner,
			want: []string{"orders:read", "orders:write", "orders:clear"},
		},
		{
			name: "staff",
			role: RoleStaff,
			want: []string{"orders:read", "orders:write"},
		},
		{name: "empty", role: "", want: nil},
		{name: "unknown", role: "admin", want: nil},
		{name: "spaces", role: " owner", want: nil},
		{name: "mixed case", role: "Owner", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.role.Permissions())
		})
	}
}

func TestRoleCanClear(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{name: "owner", role: RoleOwner, want: true},
		{name: "staff", role: RoleStaff, want: false},
		{name: "empty", role: "", want: false},
		{name: "unknown", role: "admin", want: false},
		{name: "spaces", role: " owner", want: false},
		{name: "mixed case", role: "Owner", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.role.CanClear())
		})
	}
}
