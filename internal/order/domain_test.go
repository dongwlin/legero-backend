package order

import (
	"testing"
	"time"
)

func TestNormalizeFormAppliesTakeoutDefaultsAndFiltersKidney(t *testing.T) {
	input := OrderFormInput{
		StapleTypeCode:      int16Ptr(StapleTypeRiceSheet),
		SizeCode:            SizeSmall,
		StapleAmountCode:    AdjustmentNormal,
		ExtraStapleUnits:    2,
		SelectedMeatCodes:   []int16{MeatKidney, MeatLeanPork, MeatLiver},
		GreensCode:          AdjustmentNormal,
		ScallionCode:        AdjustmentNormal,
		PepperCode:          AdjustmentNormal,
		DiningMethodCode:    DiningMethodTakeout,
		PackagingCode:       nil,
		PackagingMethodCode: nil,
		Note:                " note ",
	}

	normalized, err := NormalizeForm(input)
	if err != nil {
		t.Fatalf("NormalizeForm() error = %v", err)
	}

	if normalized.ExtraStapleUnits != 0 {
		t.Fatalf("expected extra staple units to reset, got %d", normalized.ExtraStapleUnits)
	}
	if len(normalized.SelectedMeatCodes) != 2 {
		t.Fatalf("expected kidney to be removed, got %v", normalized.SelectedMeatCodes)
	}
	if normalized.PackagingCode == nil || *normalized.PackagingCode != PackagingContainer {
		t.Fatalf("expected default container packaging, got %+v", normalized.PackagingCode)
	}
	if normalized.PackagingMethodCode == nil || *normalized.PackagingMethodCode != PackagingMethodTogether {
		t.Fatalf("expected together packaging method, got %+v", normalized.PackagingMethodCode)
	}
	if normalized.Note != "note" {
		t.Fatalf("expected note trimming, got %q", normalized.Note)
	}
}

func TestNormalizeFormPromotesRiceSmallToMedium(t *testing.T) {
	input := OrderFormInput{
		StapleTypeCode:      int16Ptr(StapleTypeRice),
		SizeCode:            SizeSmall,
		StapleAmountCode:    AdjustmentNormal,
		SelectedMeatCodes:   []int16{MeatLeanPork},
		GreensCode:          AdjustmentNormal,
		ScallionCode:        AdjustmentNormal,
		PepperCode:          AdjustmentNormal,
		DiningMethodCode:    DiningMethodDineIn,
		PackagingCode:       int16Ptr(PackagingBag),
		PackagingMethodCode: int16Ptr(PackagingMethodTogether),
	}

	normalized, err := NormalizeForm(input)
	if err != nil {
		t.Fatalf("NormalizeForm() error = %v", err)
	}

	if normalized.SizeCode != SizeMedium {
		t.Fatalf("expected rice small to normalize to medium, got %d", normalized.SizeCode)
	}
	if normalized.PackagingCode != nil || normalized.PackagingMethodCode != nil {
		t.Fatalf("expected dine-in packaging fields to be nil")
	}
}

func TestNormalizeFormRejectsEmptyOrder(t *testing.T) {
	input := OrderFormInput{
		SizeCode:          SizeSmall,
		StapleAmountCode:  AdjustmentNormal,
		SelectedMeatCodes: []int16{},
		GreensCode:        AdjustmentNormal,
		ScallionCode:      AdjustmentNormal,
		PepperCode:        AdjustmentNormal,
		DiningMethodCode:  DiningMethodDineIn,
	}

	if _, err := NormalizeForm(input); err == nil {
		t.Fatal("expected validation error for empty order")
	}
}

func TestCalculateTotalPriceCents(t *testing.T) {
	input := OrderFormInput{
		StapleTypeCode:      int16Ptr(StapleTypeYiNoodle),
		SizeCode:            SizeLarge,
		StapleAmountCode:    AdjustmentNormal,
		ExtraStapleUnits:    2,
		SelectedMeatCodes:   []int16{MeatLeanPork},
		GreensCode:          AdjustmentNormal,
		ScallionCode:        AdjustmentNormal,
		PepperCode:          AdjustmentNormal,
		DiningMethodCode:    DiningMethodTakeout,
		PackagingCode:       int16Ptr(PackagingContainer),
		PackagingMethodCode: int16Ptr(PackagingMethodTogether),
	}

	got := CalculateTotalPriceCents(input)
	want := yiNoodleLargeBasePriceCents + 2*ExtraStapleUnitPriceCents + PlasticContainerPriceCents
	if got != want {
		t.Fatalf("CalculateTotalPriceCents() = %d, want %d", got, want)
	}
}

func TestToggleStepClearsCompletedAtWhenOrderBecomesIncomplete(t *testing.T) {
	completedAt := time.Now()
	item := Order{
		StapleTypeCode:       int16Ptr(StapleTypeRiceSheet),
		SizeCode:             SizeSmall,
		StapleAmountCode:     AdjustmentNormal,
		SelectedMeatCodes:    []int16{MeatLeanPork},
		GreensCode:           AdjustmentNormal,
		ScallionCode:         AdjustmentNormal,
		PepperCode:           AdjustmentNormal,
		DiningMethodCode:     DiningMethodDineIn,
		StapleStepStatusCode: StepStatusCompleted,
		MeatStepStatusCode:   StepStatusCompleted,
		CompletedAt:          &completedAt,
	}

	toggled := ToggleStep(item, "staple")
	if toggled.StapleStepStatusCode != StepStatusNotStarted {
		t.Fatalf("expected staple step to revert to not started, got %d", toggled.StapleStepStatusCode)
	}
	if toggled.CompletedAt != nil {
		t.Fatal("expected completedAt to be cleared when order becomes incomplete")
	}
}

func int16Ptr(value int16) *int16 {
	return &value
}
