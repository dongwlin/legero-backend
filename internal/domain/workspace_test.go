package domain

import (
	"reflect"
	"testing"
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
			if got := tt.role.Valid(); got != tt.want {
				t.Fatalf("Role(%q).Valid() = %v, want %v", tt.role, got, tt.want)
			}
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
			got := tt.role.Permissions()
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Role(%q).Permissions() = %v, want %v", tt.role, got, tt.want)
			}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.CanClear(); got != tt.want {
				t.Fatalf("Role(%q).CanClear() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}
