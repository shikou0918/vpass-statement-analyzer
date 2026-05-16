package httpadapter

import (
	"time"

	"vpass-statement-analyzer/backend/internal/domain"
	"vpass-statement-analyzer/backend/internal/usecase"
)

type importFileResponse struct {
	ID             int64     `json:"id"`
	FileName       string    `json:"fileName"`
	FileHash       string    `json:"fileHash"`
	CreditCardID   *int64    `json:"creditCardId"`
	DetectedFormat string    `json:"detectedFormat"`
	HasHeader      bool      `json:"hasHeader"`
	RowCount       int       `json:"rowCount"`
	ImportedAt     time.Time `json:"importedAt"`
}

type createImportResponse struct {
	ImportFile            importFileResponse `json:"importFile"`
	ImportedCount         int                `json:"importedCount"`
	DuplicateSkippedCount int                `json:"duplicateSkippedCount"`
	ErrorCount            int                `json:"errorCount"`
}

type transactionResponse struct {
	ID                    int64     `json:"id"`
	SourceFileID          int64     `json:"sourceFileId"`
	CreditCardID          *int64    `json:"creditCardId"`
	UsageDate             time.Time `json:"usageDate"`
	MerchantName          string    `json:"merchantName"`
	CardUser              string    `json:"cardUser"`
	PaymentMethod         string    `json:"paymentMethod"`
	BillingMonth          string    `json:"billingMonth"`
	UsageAmount           *int64    `json:"usageAmount"`
	BilledAmount          *int64    `json:"billedAmount"`
	CategoryID            *int64    `json:"categoryId"`
	Memo                  string    `json:"memo"`
	ExcludedFromAnalytics bool      `json:"excludedFromAnalytics"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type categoryResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type categoryRuleResponse struct {
	ID         int64     `json:"id"`
	MatchType  string    `json:"matchType"`
	Pattern    string    `json:"pattern"`
	CategoryID int64     `json:"categoryId"`
	Priority   int       `json:"priority"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type creditCardResponse struct {
	ID          int64     `json:"id"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func importFileToResponse(item domain.ImportFile) importFileResponse {
	return importFileResponse{
		ID:             item.ID,
		FileName:       item.FileName,
		FileHash:       item.FileHash,
		CreditCardID:   item.CreditCardID,
		DetectedFormat: item.DetectedFormat,
		HasHeader:      item.HasHeader,
		RowCount:       item.RowCount,
		ImportedAt:     item.ImportedAt,
	}
}

func createImportToResponse(result usecase.CreateImportResult) createImportResponse {
	return createImportResponse{
		ImportFile:            importFileToResponse(result.ImportFile),
		ImportedCount:         result.ImportedCount,
		DuplicateSkippedCount: result.DuplicateSkippedCount,
		ErrorCount:            result.ErrorCount,
	}
}

func transactionToResponse(item domain.Transaction) transactionResponse {
	return transactionResponse{
		ID:                    item.ID,
		SourceFileID:          item.SourceFileID,
		CreditCardID:          item.CreditCardID,
		UsageDate:             item.UsageDate,
		MerchantName:          item.MerchantName,
		CardUser:              item.CardUser,
		PaymentMethod:         item.PaymentMethod,
		BillingMonth:          item.BillingMonth,
		UsageAmount:           item.UsageAmount,
		BilledAmount:          item.BilledAmount,
		CategoryID:            item.CategoryID,
		Memo:                  item.Memo,
		ExcludedFromAnalytics: item.ExcludedFromAnalytics,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func categoryToResponse(item domain.Category) categoryResponse {
	return categoryResponse{
		ID:        item.ID,
		Name:      item.Name,
		Color:     item.Color,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func categoryRuleToResponse(item domain.CategoryRule) categoryRuleResponse {
	return categoryRuleResponse{
		ID:         item.ID,
		MatchType:  item.MatchType,
		Pattern:    item.Pattern,
		CategoryID: item.CategoryID,
		Priority:   item.Priority,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

func creditCardToResponse(item domain.CreditCard) creditCardResponse {
	return creditCardResponse{
		ID:          item.ID,
		DisplayName: item.DisplayName,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapResponses[T any, R any](items []T, mapper func(T) R) []R {
	responses := make([]R, 0, len(items))
	for _, item := range items {
		responses = append(responses, mapper(item))
	}
	return responses
}
