package model

import "time"

// NeedsStapleStep reports whether the order form requires a staple cooking step.
func (f OrderFormInput) NeedsStapleStep() bool {
	return f.StapleTypeCode != nil && *f.StapleTypeCode != StapleTypeRice
}

// NeedsMeatStep reports whether the order form requires a meat cooking step.
func (f OrderFormInput) NeedsMeatStep() bool {
	return len(f.SelectedMeatCodes) > 0
}

// InitialStepStatuses returns the initial staple and meat step status codes
// for a new order based on the form input.
func (f OrderFormInput) InitialStepStatuses() (int16, int16, *time.Time) {
	stapleStatus := int16(StepStatusUnrequired)
	if f.NeedsStapleStep() {
		stapleStatus = StepStatusNotStarted
	}

	meatStatus := int16(StepStatusUnrequired)
	if f.NeedsMeatStep() {
		meatStatus = StepStatusNotStarted
	}

	return stapleStatus, meatStatus, nil
}

// CanServe reports whether an order is ready to be marked as served
// (all required steps are completed).
func (o Order) CanServe() bool {
	form := o.ToFormInput()
	if form.NeedsStapleStep() && o.StapleStepStatusCode != StepStatusCompleted {
		return false
	}
	if form.NeedsMeatStep() && o.MeatStepStatusCode != StepStatusCompleted {
		return false
	}
	return true
}

// ToggleStep toggles the completion state of a cooking step ("staple" or "meat").
// Returns the (possibly modified) order.
func (o Order) ToggleStep(step string) Order {
	switch step {
	case "staple":
		form := o.ToFormInput()
		if !form.NeedsStapleStep() || o.StapleStepStatusCode == StepStatusUnrequired {
			return o
		}
		if o.StapleStepStatusCode == StepStatusCompleted {
			o.StapleStepStatusCode = StepStatusNotStarted
		} else {
			o.StapleStepStatusCode = StepStatusCompleted
		}
	case "meat":
		if o.MeatStepStatusCode == StepStatusUnrequired {
			return o
		}
		if o.MeatStepStatusCode == StepStatusCompleted {
			o.MeatStepStatusCode = StepStatusNotStarted
		} else {
			o.MeatStepStatusCode = StepStatusCompleted
		}
	default:
		return o
	}

	if o.CompletedAt != nil && !o.CanServe() {
		o.CompletedAt = nil
	}

	return o
}

// ToggleServed toggles the served (completed) state of an order.
func (o Order) ToggleServed(now time.Time) Order {
	if o.CompletedAt == nil {
		completedAt := now
		o.CompletedAt = &completedAt
		return o
	}

	o.CompletedAt = nil
	return o
}

// ToFormInput reconstructs an OrderFormInput from an Order.
func (o Order) ToFormInput() OrderFormInput {
	return OrderFormInput{
		StapleTypeCode:       o.StapleTypeCode,
		SizeCode:             o.SizeCode,
		CustomSizePriceCents: o.CustomSizePriceCents,
		StapleAmountCode:     o.StapleAmountCode,
		ExtraStapleUnits:     o.ExtraStapleUnits,
		SelectedMeatCodes:    CloneInt16s(o.SelectedMeatCodes),
		GreensCode:           o.GreensCode,
		ScallionCode:         o.ScallionCode,
		PepperCode:           o.PepperCode,
		DiningMethodCode:     o.DiningMethodCode,
		PackagingCode:        o.PackagingCode,
		PackagingMethodCode:  o.PackagingMethodCode,
		Note:                 o.Note,
	}
}

// CloneInt16s returns a copy of the slice, or an empty slice if nil.
func CloneInt16s(values []int16) []int16 {
	if len(values) == 0 {
		return []int16{}
	}
	cloned := make([]int16, len(values))
	copy(cloned, values)
	return cloned
}
