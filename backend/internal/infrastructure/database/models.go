package database

import "time"

type ImportFileModel struct {
	ID             int64 `gorm:"primaryKey"`
	FileName       string
	FileHash       string `gorm:"uniqueIndex"`
	CreditCardID   *int64 `gorm:"index"`
	DetectedFormat string
	HasHeader      bool
	RowCount       int
	ImportedAt     time.Time
}

type ImportMappingModel struct {
	ID                int64 `gorm:"primaryKey"`
	SourceFileID      int64 `gorm:"index"`
	SourceColumnName  string
	SourceColumnIndex int
	TargetField       string
	Confidence        float64
	CreatedAt         time.Time
}

type ImportErrorModel struct {
	ID           int64 `gorm:"primaryKey"`
	SourceFileID int64 `gorm:"index"`
	RowNumber    int
	ErrorType    string
	Message      string
	RawColumns   string
	CreatedAt    time.Time
}

type TransactionModel struct {
	ID                    int64  `gorm:"primaryKey"`
	SourceFileID          int64  `gorm:"index"`
	CreditCardID          *int64 `gorm:"index"`
	UsageDate             time.Time
	MerchantName          string `gorm:"index"`
	CardUser              string
	PaymentMethod         string
	BillingMonth          string `gorm:"index"`
	UsageAmount           *int64
	BilledAmount          *int64
	CategoryID            *int64 `gorm:"index"`
	Memo                  string
	ExcludedFromAnalytics bool
	RawColumns            string
	DedupeKey             string `gorm:"uniqueIndex"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type CategoryModel struct {
	ID        int64 `gorm:"primaryKey"`
	Name      string
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRuleModel struct {
	ID         int64  `gorm:"primaryKey"`
	MatchType  string `gorm:"uniqueIndex:idx_category_rule_unique"`
	Pattern    string `gorm:"uniqueIndex:idx_category_rule_unique"`
	CategoryID int64  `gorm:"index;uniqueIndex:idx_category_rule_unique"`
	Priority   int    `gorm:"index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreditCardModel struct {
	ID          int64  `gorm:"primaryKey"`
	DisplayName string `gorm:"uniqueIndex"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
