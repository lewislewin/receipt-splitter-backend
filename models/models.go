package models

import "time"

type ParsedReceipt struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	MonzoID   string        `json:"monzo_id"`
	Reason    string        `json:"reason"`
	Items     []ReceiptItem `json:"items,omitempty"`
	Modifiers []Modifier    `json:"modifiers,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

type ReceiptItem struct {
	ID    string  `json:"id"`
	Item  string  `json:"item"`
	Price float64 `json:"price"`
	Qty   int     `json:"qty"`
}

type Modifier struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Value      float64  `json:"value"`
	Percentage *float64 `json:"percentage,omitempty"`
	Include    bool     `json:"include"`
}
