package usecase

import "vpass-statement-analyzer/backend/internal/domain"

type App struct {
	repos Repositories
	tx    TxManager
}

func NewApp(repos Repositories, tx TxManager) *App {
	return &App{repos: repos, tx: tx}
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

type ImportMappingCandidate struct {
	SourceColumnName  string   `json:"sourceColumnName,omitempty"`
	SourceColumnIndex int      `json:"sourceColumnIndex"`
	TargetField       string   `json:"targetField"`
	SampleValues      []string `json:"sampleValues"`
	Required          bool     `json:"required"`
}

type ImportPreviewRow struct {
	RowNumber  int            `json:"rowNumber"`
	Normalized map[string]any `json:"normalized"`
	RawColumns []string       `json:"rawColumns"`
}

type ImportRowError struct {
	RowNumber  int      `json:"rowNumber"`
	ErrorType  string   `json:"errorType"`
	Message    string   `json:"message"`
	RawColumns []string `json:"rawColumns"`
}

type ImportPreview struct {
	PreviewID         string                   `json:"previewId"`
	FileName          string                   `json:"fileName"`
	FileHash          string                   `json:"fileHash"`
	DetectedFormat    string                   `json:"detectedFormat"`
	Encoding          string                   `json:"encoding"`
	HasHeader         bool                     `json:"hasHeader"`
	MappingCandidates []ImportMappingCandidate `json:"mappingCandidates"`
	PreviewRows       []ImportPreviewRow       `json:"previewRows"`
	Errors            []ImportRowError         `json:"errors"`
	DuplicateFile     bool                     `json:"duplicateFile"`
}

type CreateImportInput struct {
	PreviewID        string            `json:"previewId"`
	FileHash         string            `json:"fileHash"`
	ConfirmedMapping map[string]string `json:"confirmedMapping"`
	Options          struct {
		ApplyCategoryRules bool `json:"applyCategoryRules"`
	} `json:"options"`
}

type CreateImportResult struct {
	ImportFile            domain.ImportFile `json:"importFile"`
	ImportedCount         int               `json:"importedCount"`
	DuplicateSkippedCount int               `json:"duplicateSkippedCount"`
	ErrorCount            int               `json:"errorCount"`
}

type TransactionFilter struct {
	BillingMonth    string
	UsageDateFrom   string
	UsageDateTo     string
	MerchantName    string
	CategoryID      string
	Keyword         string
	AmountMin       *int64
	AmountMax       *int64
	IncludeExcluded bool
	Page            int
	PageSize        int
	Sort            string
	Order           string
}

type UpdateTransactionInput struct {
	CategoryID            *int64
	CategoryIDSet         bool
	Memo                  *string
	ExcludedFromAnalytics *bool
}

type CategoryInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type CategoryRuleInput struct {
	MatchType  string `json:"matchType"`
	Pattern    string `json:"pattern"`
	CategoryID int64  `json:"categoryId"`
	Priority   int    `json:"priority"`
}

type SummaryFilter struct {
	Month       string
	From        string
	To          string
	FromMonth   string
	ToMonth     string
	BasisDate   string
	BasisAmount string
	Limit       int
	Merchant    string
	CategoryID  string
	MaxAmount   int64
}

type SummaryRows struct {
	TotalAmount      int64
	PreviousAmount   int64
	TransactionCount int64
	Daily            []ChartPoint
	Ranking          []RankingItem
	CategoryItems    []CategorySummaryItem
	Trend            []ChartPoint
	Recurring        []RecurringCandidate
	SmallFrequent    []SmallFrequentCandidate
}

type ChartPoint struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

type RankingItem struct {
	MerchantName     string `json:"merchantName"`
	TotalAmount      int64  `json:"totalAmount"`
	TransactionCount int64  `json:"transactionCount"`
}

type CategorySummaryItem struct {
	CategoryID       *int64  `json:"categoryId"`
	CategoryName     string  `json:"categoryName"`
	Color            string  `json:"color"`
	TotalAmount      int64   `json:"totalAmount"`
	TransactionCount int64   `json:"transactionCount"`
	Ratio            float64 `json:"ratio"`
}

type RecurringCandidate struct {
	MerchantName      string `json:"merchantName"`
	AverageAmount     int64  `json:"averageAmount"`
	OccurrenceMonths  int    `json:"occurrenceMonths"`
	LastTransactionAt string `json:"lastTransactionAt"`
}

type SmallFrequentCandidate struct {
	MerchantName     string `json:"merchantName"`
	TotalAmount      int64  `json:"totalAmount"`
	TransactionCount int64  `json:"transactionCount"`
	AverageAmount    int64  `json:"averageAmount"`
}
