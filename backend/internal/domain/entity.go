package domain

import "time"

type ImportFile struct {
	ID             int64
	FileName       string
	FileHash       string
	DetectedFormat string
	HasHeader      bool
	RowCount       int
	ImportedAt     time.Time
}

type ImportMapping struct {
	ID                int64
	SourceFileID      int64
	SourceColumnName  string
	SourceColumnIndex int
	TargetField       string
	Confidence        float64
	CreatedAt         time.Time
}

type ImportError struct {
	ID           int64
	SourceFileID int64
	RowNumber    int
	ErrorType    string
	Message      string
	RawColumns   []string
	CreatedAt    time.Time
}

type Transaction struct {
	ID                    int64
	SourceFileID          int64
	UsageDate             time.Time
	MerchantName          string
	CardUser              string
	PaymentMethod         string
	BillingMonth          string
	UsageAmount           *int64
	BilledAmount          *int64
	CategoryID            *int64
	Memo                  string
	ExcludedFromAnalytics bool
	RawColumns            []string
	DedupeKey             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Category struct {
	ID        int64
	Name      string
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRule struct {
	ID         int64
	MatchType  string
	Pattern    string
	CategoryID int64
	Priority   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AppSettings struct {
	DefaultBasisDate   string
	DefaultBasisAmount string
}
