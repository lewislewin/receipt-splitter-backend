package models

import "time"

// Modifier represents the modifiers table
type Modifier struct {
	ID         string   `json:"id"`         // UUID
	Type       string   `json:"type"`       // Modifier type (e.g., discount)
	Value      *float64 `json:"value"`      // Optional value
	Percentage *float64 `json:"percentage"` // Optional percentage
	Include    bool     `json:"include"`    // Whether to include this modifier
}

// ReceiptItem represents the receipt_items table
type ReceiptItem struct {
	ID    string   `json:"id"`    // UUID
	Item  string   `json:"item"`  // Item name
	Price *float64 `json:"price"` // Optional price
	Qty   int      `json:"qty"`   // Quantity
}

// ParsedReceipt represents the parsed_receipts table and associated relationships
type ParsedReceipt struct {
	ID        string        `json:"id"`         // UUID
	Name      string        `json:"name"`       // Receipt name
	MonzoID   string        `json:"monzo_id"`   // Associated Monzo ID
	Reason    string        `json:"reason"`     // Reason for the receipt
	Items     []ReceiptItem `json:"items"`      // Associated items
	Modifiers []Modifier    `json:"modifiers"`  // Associated modifiers
	CreatedAt time.Time     `json:"created_at"` // Timestamp of creation
}
