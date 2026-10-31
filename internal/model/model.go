package model

import "time"

type Admin struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;size:64"`
	PasswordHash string `gorm:"size:255"`
	CreatedAt    time.Time
}

type Product struct {
	ID      string `gorm:"primaryKey;size:64"`
	Name    string `gorm:"size:255"`
	TaxCode string `gorm:"size:64"`
	Active  bool   `gorm:"default:true"`
}

type ProductVariant struct {
	ID          uint   `gorm:"primaryKey"`
	ProductID   string `gorm:"index;size:64"`
	SKU         string `gorm:"uniqueIndex;size:64"`
	VariantName string `gorm:"size:64"` // 耳钩款 / 耳夹款
	PriceCents  int64
	FileKey     string `gorm:"size:512"`
	FileName    string `gorm:"size:255"`
	FileSize    int64
}

type Order struct {
	ID              string `gorm:"primaryKey;size:32"`
	UserID          string `gorm:"index;size:64"`
	Status          string `gorm:"type:varchar(20);index"`
	Currency        string `gorm:"size:3;default:USD"`
	SubtotalCents   int64
	TaxCents        int64
	TotalCents      int64
	TaxRate         float64 `gorm:"type:decimal(10,6)"`
	TaxJurisdiction string  `gorm:"size:255"`
	ToCountry       string  `gorm:"size:8"`
	ToState         string  `gorm:"size:32"`
	ToZip           string  `gorm:"size:32"`
	PayPalOrderID   string  `gorm:"index;size:64"`
	PayPalCaptureID string  `gorm:"index;size:64"`
	PaidAt          *time.Time
	DeliveredAt     *time.Time
	ClosedAt        *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Items           []OrderItem `gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
	ID             uint   `gorm:"primaryKey"`
	OrderID        string `gorm:"index;size:32"`
	ProductID      string `gorm:"size:64"`
	VariantSKU     string `gorm:"size:64"`
	ProductName    string `gorm:"size:255"`
	VariantName    string `gorm:"size:64"`
	UnitPriceCents int64
	Quantity       int
}

type PayPalWebhookLog struct {
	ID        uint   `gorm:"primaryKey"`
	Provider  string `gorm:"size:32;uniqueIndex:uk_provider_event"`
	EventID   string `gorm:"size:128;uniqueIndex:uk_provider_event"`
	EventType string `gorm:"size:64"`
	OrderID   string `gorm:"index;size:32"`
	Processed bool   `gorm:"default:false"`
	CreatedAt time.Time
}

type Refund struct {
	ID             uint   `gorm:"primaryKey"`
	OrderID        string `gorm:"index;size:32"`
	AdminID        string `gorm:"size:64"`
	AmountCents    int64
	PayPalRefundID string `gorm:"size:64"`
	Status         string `gorm:"size:20"` // PENDING / REJECTED / COMPLETED / FAILED
	RejectReason   string `gorm:"size:512"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type DownloadLog struct {
	ID         uint   `gorm:"primaryKey"`
	OrderID    string `gorm:"index;size:32"`
	VariantSKU string `gorm:"size:64"`
	IP         string `gorm:"size:64"`
	UserAgent  string `gorm:"size:512"`
	CreatedAt  time.Time
}

func AllModels() []any {
	return []any{
		&Admin{}, &Product{}, &ProductVariant{},
		&Order{}, &OrderItem{},
		&PayPalWebhookLog{}, &Refund{}, &DownloadLog{},
	}
}
