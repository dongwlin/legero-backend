package model

import "testing"

func TestNormalizePhone(t *testing.T) {
	got := NormalizePhone("+86 138-0013-8000")
	if got != "13800138000" {
		t.Fatalf("NormalizePhone() = %q, want %q", got, "13800138000")
	}
}
