package market

type Bar struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type IndicatorPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type IndicatorMultiPoint struct {
	Time   int64              `json:"time"`
	Values map[string]float64 `json:"values"`
}
