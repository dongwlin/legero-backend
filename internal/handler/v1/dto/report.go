package dto

// ReportResponse is the extensible period-report response. StartDate and
// EndDate are inclusive business-calendar dates; the service keeps the
// corresponding half-open absolute window internally.
type ReportResponse struct {
	Period    string        `json:"period"`
	StartDate string        `json:"startDate"`
	EndDate   string        `json:"endDate"`
	Metrics   ReportMetrics `json:"metrics"`
}

// ReportMetrics contains the M1 completed-order metrics.
type ReportMetrics struct {
	RevenueCents              int                  `json:"revenueCents"`
	CompletedOrderCount       int                  `json:"completedOrderCount"`
	AverageOrderValueCents    int                  `json:"averageOrderValueCents"`
	AveragePreparationSeconds int                  `json:"averagePreparationSeconds"`
	Peak30MinuteBuckets       []Peak30Minute       `json:"peak30MinuteBuckets"`
	StapleSales               []StapleSale         `json:"stapleSales"`
	NoStapleOrderCount        int                  `json:"noStapleOrderCount"`
	UnknownStapleOrderCount   int                  `json:"unknownStapleOrderCount"`
	StandardSize              StandardSizeMetrics  `json:"standardSize"`
	TotalFriedEggCount        int                  `json:"totalFriedEggCount"`
	Takeout                   RatioMetric          `json:"takeout"`
	Customizations            CustomizationMetrics `json:"customizations"`
}

// RatioMetric contains a metric numerator, denominator and ratio in [0, 1].
type RatioMetric struct {
	Count       int     `json:"count"`
	Denominator int     `json:"denominator"`
	Ratio       float64 `json:"ratio"`
}

// Peak30Minute is the busiest completed_at half-hour bucket.
type Peak30Minute struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	OrderCount int    `json:"orderCount"`
}

// StapleSale is a fixed staple-code count entry.
type StapleSale struct {
	StapleTypeCode int16 `json:"stapleTypeCode"`
	OrderCount     int   `json:"orderCount"`
}

// StandardSizeMetrics reports standard sizes separately and keeps custom
// orders out of their ratio denominator.
type StandardSizeMetrics struct {
	StandardCount        int         `json:"standardCount"`
	CustomSizeOrderCount int         `json:"customSizeOrderCount"`
	Small                RatioMetric `json:"small"`
	Medium               RatioMetric `json:"medium"`
	Large                RatioMetric `json:"large"`
}

// CustomizationMetrics reports overlapping customization predicates and their
// unique union.
type CustomizationMetrics struct {
	LeanMeatOnly RatioMetric `json:"leanMeatOnly"`
	NoIntestine  RatioMetric `json:"noIntestine"`
	Union        RatioMetric `json:"union"`
}
