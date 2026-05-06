package database

import (
	"encoding/json"

	"vpass-statement-analyzer/backend/internal/domain"
)

func importFileToDomain(m ImportFileModel) domain.ImportFile {
	return domain.ImportFile{ID: m.ID, FileName: m.FileName, FileHash: m.FileHash, DetectedFormat: m.DetectedFormat, HasHeader: m.HasHeader, RowCount: m.RowCount, ImportedAt: m.ImportedAt}
}

func transactionToDomain(m TransactionModel) domain.Transaction {
	var raw []string
	_ = json.Unmarshal([]byte(m.RawColumns), &raw)
	return domain.Transaction{
		ID:                    m.ID,
		SourceFileID:          m.SourceFileID,
		UsageDate:             m.UsageDate,
		MerchantName:          m.MerchantName,
		CardUser:              m.CardUser,
		PaymentMethod:         m.PaymentMethod,
		BillingMonth:          m.BillingMonth,
		UsageAmount:           m.UsageAmount,
		BilledAmount:          m.BilledAmount,
		CategoryID:            m.CategoryID,
		Memo:                  m.Memo,
		ExcludedFromAnalytics: m.ExcludedFromAnalytics,
		RawColumns:            raw,
		DedupeKey:             m.DedupeKey,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

func transactionToModel(t domain.Transaction) TransactionModel {
	raw, _ := json.Marshal(t.RawColumns)
	return TransactionModel{
		ID:                    t.ID,
		SourceFileID:          t.SourceFileID,
		UsageDate:             t.UsageDate,
		MerchantName:          t.MerchantName,
		CardUser:              t.CardUser,
		PaymentMethod:         t.PaymentMethod,
		BillingMonth:          t.BillingMonth,
		UsageAmount:           t.UsageAmount,
		BilledAmount:          t.BilledAmount,
		CategoryID:            t.CategoryID,
		Memo:                  t.Memo,
		ExcludedFromAnalytics: t.ExcludedFromAnalytics,
		RawColumns:            string(raw),
		DedupeKey:             t.DedupeKey,
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
	}
}

func categoryToDomain(m CategoryModel) domain.Category {
	return domain.Category{ID: m.ID, Name: m.Name, Color: m.Color, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}

func ruleToDomain(m CategoryRuleModel) domain.CategoryRule {
	return domain.CategoryRule{ID: m.ID, MatchType: m.MatchType, Pattern: m.Pattern, CategoryID: m.CategoryID, Priority: m.Priority, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
