package dto

// DailyItem is a single day in the daily statistics result.
type DailyItem struct {
	Date            string `json:"date"`
	OrderCount      int    `json:"orderCount"`
	TotalPriceCents int    `json:"totalPriceCents"`
}

// DailyResponseData is the daily statistics response.
type DailyResponseData struct {
	Items []DailyItem `json:"items"`
}
