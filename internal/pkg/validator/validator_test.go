package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPhoneNumValidation(t *testing.T) {

	testCases := []struct {
		name        string
		phoneNumber string
		wantValid   bool
	}{
		{
			name:        "Valid 11-digit number",
			phoneNumber: "13812345678",
			wantValid:   true,
		},
		{
			name:        "Valid new number starting with 19",
			phoneNumber: "19999999999",
			wantValid:   true,
		},
		{
			name:        "Invalid short length",
			phoneNumber: "1381234567",
			wantValid:   false,
		},
		{
			name:        "Invalid long length",
			phoneNumber: "138123456789",
			wantValid:   false,
		},
		{
			name:        "Invalid contains letter",
			phoneNumber: "1381234567a",
			wantValid:   false,
		},
		{
			name:        "Invalid contains symbol",
			phoneNumber: "138-1234-5678",
			wantValid:   false,
		},
		{
			name:        "Invalid starts with 2",
			phoneNumber: "23812345678",
			wantValid:   false,
		},
		{
			name:        "Invalid empty string",
			phoneNumber: "",
			wantValid:   false,
		},
		{
			name:        "Invalid international format",
			phoneNumber: "+8613812345678",
			wantValid:   false,
		},
		{
			name:        "Invalid with country code prefix",
			phoneNumber: "8613812345678",
			wantValid:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			err := Validate.Var(tc.phoneNumber, "phone_num")

			if tc.wantValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
