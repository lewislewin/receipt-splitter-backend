package models

import "time"

// Receipt represents a receipt with associated items and modifiers
type Receipt struct {
	ID        string        `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string        `gorm:"not null" json:"name"`
	MonzoID   string        `gorm:"not null" json:"monzo_id"`
	Reason    string        `gorm:"type:text" json:"reason"`
	UserID    string        `gorm:"not null" json:"-"`
	User      User          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user"`
	Items     []ReceiptItem `gorm:"foreignKey:ReceiptID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	Modifiers []Modifier    `gorm:"foreignKey:ReceiptID;constraint:OnDelete:CASCADE" json:"modifiers,omitempty"`
	CreatedAt time.Time     `gorm:"autoCreateTime" json:"created_at"`
}

// ReceiptItem represents an item on a receipt
type ReceiptItem struct {
	ID        string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ReceiptID string  `gorm:"not null" json:"-"`
	Item      string  `gorm:"not null" json:"item"`
	Price     float64 `gorm:"not null" json:"price"`
	Qty       int     `gorm:"not null" json:"qty"`
}

// Modifier represents a discount or adjustment applied to a receipt
type Modifier struct {
	ID         string   `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ReceiptID  string   `gorm:"not null" json:"-"`
	Type       string   `gorm:"not null" json:"type"`
	Value      float64  `gorm:"not null" json:"value"`
	Percentage *float64 `gorm:"type:numeric" json:"percentage,omitempty"`
	Include    bool     `gorm:"not null" json:"include"`
}

// User represents a system user
type User struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"unique;not null" json:"email"`
	Password  string    `gorm:"not null" json:"password"`
	MonzoID   string    `gorm:"not null" json:"monzo_id"`
	Receipts  []Receipt `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"receipts,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
