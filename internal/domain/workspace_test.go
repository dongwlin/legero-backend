package domain

import "testing"

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
