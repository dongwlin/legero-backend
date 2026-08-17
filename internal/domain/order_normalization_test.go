package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestOrderFormInput_Normalize(t *testing.T) {
	t.Run("valid input passes normalization", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if result.Note != "" {
			t.Errorf("Normalize() Note = %q, want empty", result.Note)
		}
	})

	t.Run("negative extra staple units returns error", func(t *testing.T) {
		input := OrderFormInput{
			ExtraStapleUnits:  -1,
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
		}

		_, err := input.Normalize()
		if !errors.Is(err, ErrNegativeExtraStapleUnits) {
			t.Errorf("Normalize() error = %v, want %v", err, ErrNegativeExtraStapleUnits)
		}
	})

	t.Run("invalid size code returns error", func(t *testing.T) {
		input := OrderFormInput{
			SizeCode:          99,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
		}

		_, err := input.Normalize()
		if !errors.Is(err, ErrInvalidSizeCode) {
			t.Errorf("Normalize() error = %v, want %v", err, ErrInvalidSizeCode)
		}
	})

	t.Run("rice small promoted to medium", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRice),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if result.SizeCode != SizeMedium {
			t.Errorf("Normalize() SizeCode = %v, want %v", result.SizeCode, SizeMedium)
		}
	})

	t.Run("empty order returns error", func(t *testing.T) {
		input := OrderFormInput{
			SizeCode:         SizeSmall,
			StapleAmountCode: AdjustmentNormal,
			GreensCode:       AdjustmentNormal,
			ScallionCode:     AdjustmentNormal,
			PepperCode:       AdjustmentNormal,
			DiningMethodCode: DiningMethodDineIn,
		}

		_, err := input.Normalize()
		if !errors.Is(err, ErrEmptyOrder) {
			t.Errorf("Normalize() error = %v, want %v", err, ErrEmptyOrder)
		}
	})

	t.Run("note is trimmed", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
			Note:              "  test note  ",
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if result.Note != "test note" {
			t.Errorf("Normalize() Note = %q, want %q", result.Note, "test note")
		}
	})

	t.Run("maximum note size is accepted", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
			Note:              strings.Repeat("n", MaxOrderNoteBytes),
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if len(result.Note) != MaxOrderNoteBytes {
			t.Fatalf("Normalize() note bytes = %d, want %d", len(result.Note), MaxOrderNoteBytes)
		}
	})

	t.Run("maximum note size is measured in UTF-8 bytes", func(t *testing.T) {
		base := strings.Repeat("界", MaxOrderNoteBytes/len("界"))
		atLimit := base + "x"
		if len(atLimit) != MaxOrderNoteBytes {
			t.Fatalf("test note bytes = %d, want %d", len(atLimit), MaxOrderNoteBytes)
		}
		if len([]rune(atLimit)) >= MaxOrderNoteBytes {
			t.Fatalf("test note should contain fewer runes than bytes: %d runes", len([]rune(atLimit)))
		}

		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
			Note:              atLimit,
		}
		if _, err := input.Normalize(); err != nil {
			t.Fatalf("Normalize() at byte limit error = %v", err)
		}

		input.Note += "x"
		if _, err := input.Normalize(); !errors.Is(err, ErrNoteTooLong) {
			t.Fatalf("Normalize() over byte limit error = %v, want %v", err, ErrNoteTooLong)
		}
	})

	t.Run("note over maximum size returns error", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
			Note:              strings.Repeat("n", MaxOrderNoteBytes+1),
		}

		_, err := input.Normalize()
		if !errors.Is(err, ErrNoteTooLong) {
			t.Fatalf("Normalize() error = %v, want %v", err, ErrNoteTooLong)
		}
	})

	t.Run("takeout sets default packaging", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodTakeout,
			SelectedMeatCodes: []int16{MeatLeanPork},
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if result.PackagingCode == nil || *result.PackagingCode != PackagingContainer {
			t.Errorf("Normalize() PackagingCode = %v, want %v", result.PackagingCode, PackagingContainer)
		}
		if result.PackagingMethodCode == nil || *result.PackagingMethodCode != PackagingMethodTogether {
			t.Errorf("Normalize() PackagingMethodCode = %v, want %v", result.PackagingMethodCode, PackagingMethodTogether)
		}
	})

	t.Run("dine-in clears packaging", func(t *testing.T) {
		packagingCode := int16(PackagingContainer)
		packagingMethod := int16(PackagingMethodTogether)
		input := OrderFormInput{
			StapleTypeCode:      int16Ptr(StapleTypeRiceSheet),
			SizeCode:            SizeSmall,
			StapleAmountCode:    AdjustmentNormal,
			GreensCode:          AdjustmentNormal,
			ScallionCode:        AdjustmentNormal,
			PepperCode:          AdjustmentNormal,
			DiningMethodCode:    DiningMethodDineIn,
			SelectedMeatCodes:   []int16{MeatLeanPork},
			PackagingCode:       &packagingCode,
			PackagingMethodCode: &packagingMethod,
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if result.PackagingCode != nil {
			t.Errorf("Normalize() PackagingCode = %v, want nil", result.PackagingCode)
		}
		if result.PackagingMethodCode != nil {
			t.Errorf("Normalize() PackagingMethodCode = %v, want nil", result.PackagingMethodCode)
		}
	})

	t.Run("small size filters out kidney", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatKidney, MeatLeanPork, MeatLiver},
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if len(result.SelectedMeatCodes) != 2 {
			t.Fatalf("Normalize() SelectedMeatCodes length = %d, want 2", len(result.SelectedMeatCodes))
		}
		if result.SelectedMeatCodes[0] != MeatLeanPork || result.SelectedMeatCodes[1] != MeatLiver {
			t.Errorf("Normalize() SelectedMeatCodes = %v, want [%d, %d]", result.SelectedMeatCodes, MeatLeanPork, MeatLiver)
		}
	})

	t.Run("medium size preserves kidney", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeMedium,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatKidney, MeatLeanPork, MeatLiver},
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if len(result.SelectedMeatCodes) != 3 {
			t.Fatalf("Normalize() SelectedMeatCodes length = %d, want 3", len(result.SelectedMeatCodes))
		}
		want := []int16{MeatLeanPork, MeatLiver, MeatKidney}
		for i, code := range want {
			if result.SelectedMeatCodes[i] != code {
				t.Errorf("Normalize() SelectedMeatCodes[%d] = %d, want %d", i, result.SelectedMeatCodes[i], code)
			}
		}
	})

	t.Run("non-yi-noodle resets extra staple units", func(t *testing.T) {
		input := OrderFormInput{
			StapleTypeCode:    int16Ptr(StapleTypeRiceSheet),
			SizeCode:          SizeSmall,
			StapleAmountCode:  AdjustmentNormal,
			GreensCode:        AdjustmentNormal,
			ScallionCode:      AdjustmentNormal,
			PepperCode:        AdjustmentNormal,
			DiningMethodCode:  DiningMethodDineIn,
			SelectedMeatCodes: []int16{MeatLeanPork},
			ExtraStapleUnits:  3,
		}

		result, err := input.Normalize()
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if result.ExtraStapleUnits != 0 {
			t.Errorf("Normalize() ExtraStapleUnits = %d, want 0", result.ExtraStapleUnits)
		}
	})
}
