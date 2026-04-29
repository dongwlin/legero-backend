package order

import "time"

func NeedsStapleStep(input OrderFormInput) bool {
	return input.StapleTypeCode != nil && *input.StapleTypeCode != StapleTypeRice
}

func NeedsMeatStep(input OrderFormInput) bool {
	return len(input.SelectedMeatCodes) > 0
}

func InitialStepStatuses(input OrderFormInput) (int16, int16, *time.Time) {
	stapleStatus := int16(StepStatusUnrequired)
	if NeedsStapleStep(input) {
		stapleStatus = StepStatusNotStarted
	}

	meatStatus := int16(StepStatusUnrequired)
	if NeedsMeatStep(input) {
		meatStatus = StepStatusNotStarted
	}

	return stapleStatus, meatStatus, nil
}

func CanServe(item Order) bool {
	if NeedsStapleStep(orderToFormInput(item)) && item.StapleStepStatusCode != StepStatusCompleted {
		return false
	}
	if NeedsMeatStep(orderToFormInput(item)) && item.MeatStepStatusCode != StepStatusCompleted {
		return false
	}
	return true
}

func ToggleStep(item Order, step string) Order {
	switch step {
	case "staple":
		if !NeedsStapleStep(orderToFormInput(item)) || item.StapleStepStatusCode == StepStatusUnrequired {
			return item
		}
		if item.StapleStepStatusCode == StepStatusCompleted {
			item.StapleStepStatusCode = StepStatusNotStarted
		} else {
			item.StapleStepStatusCode = StepStatusCompleted
		}
	case "meat":
		if item.MeatStepStatusCode == StepStatusUnrequired {
			return item
		}
		if item.MeatStepStatusCode == StepStatusCompleted {
			item.MeatStepStatusCode = StepStatusNotStarted
		} else {
			item.MeatStepStatusCode = StepStatusCompleted
		}
	default:
		return item
	}

	if item.CompletedAt != nil && !CanServe(item) {
		item.CompletedAt = nil
	}

	return item
}

func ToggleServed(item Order, now time.Time) Order {
	if item.CompletedAt == nil {
		completedAt := now
		item.CompletedAt = &completedAt
		return item
	}

	item.CompletedAt = nil
	return item
}

func orderToFormInput(item Order) OrderFormInput {
	return OrderFormInput{
		StapleTypeCode:       item.StapleTypeCode,
		SizeCode:             item.SizeCode,
		CustomSizePriceCents: item.CustomSizePriceCents,
		StapleAmountCode:     item.StapleAmountCode,
		ExtraStapleUnits:     item.ExtraStapleUnits,
		SelectedMeatCodes:    cloneInt16s(item.SelectedMeatCodes),
		GreensCode:           item.GreensCode,
		ScallionCode:         item.ScallionCode,
		PepperCode:           item.PepperCode,
		DiningMethodCode:     item.DiningMethodCode,
		PackagingCode:        item.PackagingCode,
		PackagingMethodCode:  item.PackagingMethodCode,
		Note:                 item.Note,
	}
}
