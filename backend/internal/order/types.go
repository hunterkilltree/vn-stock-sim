package order

import "time"

type Order struct {
	ID          string  `json:"id"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Type        string  `json:"type"`
	Quantity    int64   `json:"quantity"`
	Status      string  `json:"status"`
	FilledPrice float64 `json:"filledPrice,omitempty"`
	FilledAt    string  `json:"filledAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

type createRequest struct {
	Symbol   string `json:"symbol" binding:"required"`
	Side     string `json:"side" binding:"required,oneof=buy sell"`
	Type     string `json:"type" binding:"required,oneof=market limit"`
	Quantity int64  `json:"quantity" binding:"required,gt=0"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
