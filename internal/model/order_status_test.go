package model

import (
	"testing"
	"time"
)

func TestOrderFormInput_NeedsStapleStep(t *testing.T) {
	tests := []struct {
		name string
		input OrderFormInput
		want bool
	}{
		{
			name: "rice sheet needs staple step",
			input: OrderFormInput{StapleTypeCode: int16Ptr(StapleTypeRiceSheet)},
			want: true,
		},
		{
			name: "rice does not need staple step",
			input: OrderFormInput{StapleTypeCode: int16Ptr(StapleTypeRice)},
			want: false,
		},
		{
			name: "nil staple type does not need staple step",
			input: OrderFormInput{StapleTypeCode: nil},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.NeedsStapleStep(); got != tt.want {
				t.Errorf("NeedsStapleStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderFormInput_NeedsMeatStep(t *testing.T) {
	tests := []struct {
		name string
		input OrderFormInput
		want bool
	}{
		{
			name: "with meat codes needs meat step",
			input: OrderFormInput{SelectedMeatCodes: []int16{MeatLeanPork}},
			want: true,
		},
		{
			name: "empty meat codes does not need meat step",
			input: OrderFormInput{SelectedMeatCodes: []int16{}},
			want: false,
		},
		{
			name: "nil meat codes does not need meat step",
			input: OrderFormInput{SelectedMeatCodes: nil},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.NeedsMeatStep(); got != tt.want {
				t.Errorf("NeedsMeatStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrder_CanServe(t *testing.T) {
	completedAt := time.Now()

	tests := []struct {
		name string
		order Order
		want bool
	}{
		{
			name: "order with no required steps can be served",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusUnrequired,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			want: true,
		},
		{
			name: "order with completed staple step can be served",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRiceSheet),
				StapleStepStatusCode: StepStatusCompleted,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			want: true,
		},
		{
			name: "order with incomplete staple step cannot be served",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRiceSheet),
				StapleStepStatusCode: StepStatusNotStarted,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			want: false,
		},
		{
			name: "order with completed meat step can be served",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusUnrequired,
				MeatStepStatusCode:   StepStatusCompleted,
				SelectedMeatCodes:    []int16{MeatLeanPork},
			},
			want: true,
		},
		{
			name: "order with incomplete meat step cannot be served",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusUnrequired,
				MeatStepStatusCode:   StepStatusNotStarted,
				SelectedMeatCodes:    []int16{MeatLeanPork},
			},
			want: false,
		},
		{
			name: "already served order can be served",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusUnrequired,
				MeatStepStatusCode:   StepStatusUnrequired,
				CompletedAt:          &completedAt,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.order.CanServe(); got != tt.want {
				t.Errorf("CanServe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrder_ToggleStep(t *testing.T) {
	completedAt := time.Now()

	tests := []struct {
		name string
		order Order
		step string
		want int16
	}{
		{
			name: "toggle staple step from not started to completed",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRiceSheet),
				StapleStepStatusCode: StepStatusNotStarted,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			step: "staple",
			want: StepStatusCompleted,
		},
		{
			name: "toggle staple step from completed to not started",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRiceSheet),
				StapleStepStatusCode: StepStatusCompleted,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			step: "staple",
			want: StepStatusNotStarted,
		},
		{
			name: "toggle meat step from not started to completed",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusUnrequired,
				MeatStepStatusCode:   StepStatusNotStarted,
			},
			step: "meat",
			want: StepStatusCompleted,
		},
		{
			name: "toggle staple step for rice is no-op",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusNotStarted,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			step: "staple",
			want: StepStatusNotStarted,
		},
		{
			name: "toggle unrequired step is no-op",
			order: Order{
				StapleTypeCode:       int16Ptr(StapleTypeRice),
				StapleStepStatusCode: StepStatusUnrequired,
				MeatStepStatusCode:   StepStatusUnrequired,
			},
			step: "meat",
			want: StepStatusUnrequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.order.ToggleStep(tt.step)
			if result.StapleStepStatusCode != tt.want && result.MeatStepStatusCode != tt.want {
				t.Errorf("ToggleStep() status = %v/%v, want %v", result.StapleStepStatusCode, result.MeatStepStatusCode, tt.want)
			}
		})
	}

	t.Run("toggle step clears completed at when order becomes incomplete", func(t *testing.T) {
		order := Order{
			StapleTypeCode:       int16Ptr(StapleTypeRiceSheet),
			StapleStepStatusCode: StepStatusCompleted,
			MeatStepStatusCode:   StepStatusCompleted,
			CompletedAt:          &completedAt,
		}

		result := order.ToggleStep("staple")
		if result.CompletedAt != nil {
			t.Error("ToggleStep() should clear CompletedAt when order becomes incomplete")
		}
	})
}

func TestOrder_ToggleServed(t *testing.T) {
	now := time.Now()

	t.Run("toggle served from nil to now", func(t *testing.T) {
		order := Order{CompletedAt: nil}
		result := order.ToggleServed(now)
		if result.CompletedAt == nil {
			t.Error("ToggleServed() should set CompletedAt")
		}
		if !result.CompletedAt.Equal(now) {
			t.Errorf("ToggleServed() CompletedAt = %v, want %v", result.CompletedAt, now)
		}
	})

	t.Run("toggle served from time to nil", func(t *testing.T) {
		order := Order{CompletedAt: &now}
		result := order.ToggleServed(now)
		if result.CompletedAt != nil {
			t.Error("ToggleServed() should clear CompletedAt")
		}
	})
}

func int16Ptr(v int16) *int16 {
	return &v
}
