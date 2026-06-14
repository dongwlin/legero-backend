package model

import "time"

// DailyRow is a single row in the daily statistics result.
type DailyRow struct {
	Date            time.Time
	OrderCount      int
	TotalPriceCents int
}
