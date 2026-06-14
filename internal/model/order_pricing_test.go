package model

import "testing"

func intPtr(v int) *int { return &v }

func TestGetBasePriceCents(t *testing.T) {
	t.Run("default (non-rice, non-yiNoodle) sizes", func(t *testing.T) {
		tests := []struct {
			name     string
			sizeCode int16
			want     int
		}{
			{"small", SizeSmall, 1000},
			{"medium", SizeMedium, 1200},
			{"large", SizeLarge, 1500},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := OrderFormInput{SizeCode: tt.sizeCode}
				if got := f.GetBasePriceCents(); got != tt.want {
					t.Errorf("GetBasePriceCents() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("yiNoodle sizes", func(t *testing.T) {
		tests := []struct {
			name     string
			sizeCode int16
			want     int
		}{
			{"small", SizeSmall, 1100},
			{"medium", SizeMedium, 1300},
			{"large", SizeLarge, 1600},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := OrderFormInput{
					StapleTypeCode: int16Ptr(StapleTypeYiNoodle),
					SizeCode:       tt.sizeCode,
				}
				if got := f.GetBasePriceCents(); got != tt.want {
					t.Errorf("GetBasePriceCents() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("rice sizes", func(t *testing.T) {
		tests := []struct {
			name     string
			sizeCode int16
			want     int
		}{
			{"medium", SizeMedium, 1500},
			{"large", SizeLarge, 2000},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := OrderFormInput{
					StapleTypeCode: int16Ptr(StapleTypeRice),
					SizeCode:       tt.sizeCode,
				}
				if got := f.GetBasePriceCents(); got != tt.want {
					t.Errorf("GetBasePriceCents() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("custom with valid price", func(t *testing.T) {
		f := OrderFormInput{
			SizeCode:             SizeCustom,
			CustomSizePriceCents: intPtr(2500),
		}
		if got := f.GetBasePriceCents(); got != 2500 {
			t.Errorf("GetBasePriceCents() = %d, want 2500", got)
		}
	})

	t.Run("custom with nil price", func(t *testing.T) {
		f := OrderFormInput{SizeCode: SizeCustom}
		if got := f.GetBasePriceCents(); got != 0 {
			t.Errorf("GetBasePriceCents() = %d, want 0", got)
		}
	})

	t.Run("custom with negative price", func(t *testing.T) {
		f := OrderFormInput{
			SizeCode:             SizeCustom,
			CustomSizePriceCents: intPtr(-100),
		}
		if got := f.GetBasePriceCents(); got != 0 {
			t.Errorf("GetBasePriceCents() = %d, want 0", got)
		}
	})

	t.Run("unknown size code returns 0", func(t *testing.T) {
		f := OrderFormInput{SizeCode: 99}
		if got := f.GetBasePriceCents(); got != 0 {
			t.Errorf("GetBasePriceCents() = %d, want 0", got)
		}
	})
}

func TestCalculateTotalPriceCents(t *testing.T) {
	t.Run("yiNoodle large with all extras and takeout container", func(t *testing.T) {
		// 1600 + 2*300 + 1*200 + 2*200 + 50 = 2850
		f := OrderFormInput{
			StapleTypeCode:   int16Ptr(StapleTypeYiNoodle),
			SizeCode:         SizeLarge,
			ExtraStapleUnits: 2,
			FriedEggCount:    1,
			TofuSkewerCount:  2,
			DiningMethodCode: DiningMethodTakeout,
			PackagingCode:    int16Ptr(PackagingContainer),
		}
		if got := f.CalculateTotalPriceCents(); got != 2850 {
			t.Errorf("CalculateTotalPriceCents() = %d, want 2850", got)
		}
	})

	t.Run("basic order with no extras", func(t *testing.T) {
		f := OrderFormInput{
			SizeCode:         SizeMedium,
			DiningMethodCode: DiningMethodDineIn,
		}
		if got := f.CalculateTotalPriceCents(); got != 1200 {
			t.Errorf("CalculateTotalPriceCents() = %d, want 1200", got)
		}
	})

	t.Run("non-yiNoodle with ExtraStapleUnits ignores extra staple cost", func(t *testing.T) {
		f := OrderFormInput{
			StapleTypeCode:   int16Ptr(StapleTypeRiceSheet),
			SizeCode:         SizeMedium,
			ExtraStapleUnits: 5,
			DiningMethodCode: DiningMethodDineIn,
		}
		// base 1200 + no extra staple + no packaging = 1200
		if got := f.CalculateTotalPriceCents(); got != 1200 {
			t.Errorf("CalculateTotalPriceCents() = %d, want 1200", got)
		}
	})

	t.Run("takeout with container packaging adds 50", func(t *testing.T) {
		f := OrderFormInput{
			SizeCode:         SizeSmall,
			DiningMethodCode: DiningMethodTakeout,
			PackagingCode:    int16Ptr(PackagingContainer),
		}
		// 1000 + 50 = 1050
		if got := f.CalculateTotalPriceCents(); got != 1050 {
			t.Errorf("CalculateTotalPriceCents() = %d, want 1050", got)
		}
	})

	t.Run("takeout with bag packaging does NOT add 50", func(t *testing.T) {
		f := OrderFormInput{
			SizeCode:         SizeSmall,
			DiningMethodCode: DiningMethodTakeout,
			PackagingCode:    int16Ptr(PackagingBag),
		}
		if got := f.CalculateTotalPriceCents(); got != 1000 {
			t.Errorf("CalculateTotalPriceCents() = %d, want 1000", got)
		}
	})

	t.Run("dine-in does NOT add packaging cost", func(t *testing.T) {
		f := OrderFormInput{
			SizeCode:         SizeSmall,
			DiningMethodCode: DiningMethodDineIn,
			PackagingCode:    int16Ptr(PackagingContainer),
		}
		if got := f.CalculateTotalPriceCents(); got != 1000 {
			t.Errorf("CalculateTotalPriceCents() = %d, want 1000", got)
		}
	})
}
