package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vpass-statement-analyzer/backend/internal/domain"
	"vpass-statement-analyzer/backend/internal/usecase"
)

type transactionRepository struct {
	db *gorm.DB
}

func (r transactionRepository) CreateMany(ctx context.Context, txs []domain.Transaction) (int, int, error) {
	created := 0
	skipped := 0
	for _, tx := range txs {
		model := transactionToModel(tx)
		res := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedupe_key"}}, DoNothing: true}).Create(&model)
		if res.Error != nil {
			return created, skipped, res.Error
		}
		if res.RowsAffected == 0 {
			skipped++
		} else {
			created++
		}
	}
	return created, skipped, nil
}

func (r transactionRepository) List(ctx context.Context, f usecase.TransactionFilter) ([]domain.Transaction, int64, error) {
	query := applyTransactionFilter(r.db.WithContext(ctx).Model(&TransactionModel{}), f)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sort := mapSort(f.Sort)
	order := "desc"
	if f.Order == "asc" {
		order = "asc"
	}
	var models []TransactionModel
	if err := query.Order(fmt.Sprintf("%s %s", sort, order)).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.Transaction, 0, len(models))
	for _, m := range models {
		items = append(items, transactionToDomain(m))
	}
	return items, total, nil
}

func (r transactionRepository) FindByID(ctx context.Context, id int64) (*domain.Transaction, error) {
	var model TransactionModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, usecase.NotFound("明細が見つかりません")
	}
	if err != nil {
		return nil, err
	}
	tx := transactionToDomain(model)
	return &tx, nil
}

func (r transactionRepository) UpdateEditable(ctx context.Context, id int64, in usecase.UpdateTransactionInput) (*domain.Transaction, error) {
	updates := map[string]any{"updated_at": time.Now()}
	if in.CategoryIDSet {
		updates["category_id"] = in.CategoryID
	}
	if in.Memo != nil {
		updates["memo"] = *in.Memo
	}
	if in.ExcludedFromAnalytics != nil {
		updates["excluded_from_analytics"] = *in.ExcludedFromAnalytics
	}
	res := r.db.WithContext(ctx).Model(&TransactionModel{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, usecase.NotFound("明細が見つかりません")
	}
	return r.FindByID(ctx, id)
}

func (r transactionRepository) DeleteByImportID(ctx context.Context, importID int64) error {
	return r.db.WithContext(ctx).Where("source_file_id = ?", importID).Delete(&TransactionModel{}).Error
}

func (r transactionRepository) Summary(ctx context.Context, f usecase.SummaryFilter) (usecase.SummaryRows, error) {
	rows := usecase.SummaryRows{}
	amountColumn := "billed_amount"
	if f.BasisAmount == "usageAmount" {
		amountColumn = "usage_amount"
	}

	baseQuery := func() *gorm.DB {
		base := r.db.WithContext(ctx).Model(&TransactionModel{}).Where("excluded_from_analytics = ?", false)
		if f.Month != "" {
			base = base.Where("billing_month = ?", f.Month)
		}
		if f.From != "" {
			base = base.Where("date(usage_date) >= ?", f.From)
		}
		if f.To != "" {
			base = base.Where("date(usage_date) <= ?", f.To)
		}
		if f.FromMonth != "" {
			base = base.Where("billing_month >= ?", f.FromMonth)
		}
		if f.ToMonth != "" {
			base = base.Where("billing_month <= ?", f.ToMonth)
		}
		if f.Merchant != "" {
			base = base.Where("merchant_name = ?", f.Merchant)
		}
		if f.CategoryID != "" {
			base = base.Where("category_id = ?", f.CategoryID)
		}
		return base
	}

	var totals struct {
		Total int64
		Count int64
	}
	_ = baseQuery().Select(fmt.Sprintf("coalesce(sum(%s), 0) as total, count(*) as count", amountColumn)).Scan(&totals).Error
	rows.TotalAmount = totals.Total
	rows.TransactionCount = totals.Count

	var daily []struct {
		Label  string
		Amount int64
	}
	_ = baseQuery().Select(fmt.Sprintf("date(usage_date) as label, coalesce(sum(%s), 0) as amount", amountColumn)).Group("date(usage_date)").Order("label asc").Scan(&daily).Error
	for _, d := range daily {
		rows.Daily = append(rows.Daily, usecase.ChartPoint{Label: d.Label, Amount: d.Amount})
	}

	var ranking []struct {
		MerchantName string
		TotalAmount  int64
		Count        int64
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	_ = baseQuery().Select(fmt.Sprintf("merchant_name as merchant_name, coalesce(sum(%s), 0) as total_amount, count(*) as count", amountColumn)).Group("merchant_name").Order("total_amount desc").Limit(limit).Scan(&ranking).Error
	for _, row := range ranking {
		rows.Ranking = append(rows.Ranking, usecase.RankingItem{MerchantName: row.MerchantName, TotalAmount: row.TotalAmount, TransactionCount: row.Count})
	}

	var categories []struct {
		CategoryID   *int64
		CategoryName string
		Color        string
		TotalAmount  int64
		Count        int64
	}
	_ = baseQuery().Joins("left join category_models on category_models.id = transaction_models.category_id").
		Select(fmt.Sprintf("transaction_models.category_id as category_id, coalesce(category_models.name, '未分類') as category_name, coalesce(category_models.color, '#9ca3af') as color, coalesce(sum(%s), 0) as total_amount, count(*) as count", amountColumn)).
		Group("transaction_models.category_id, category_models.name, category_models.color").
		Order("total_amount desc").Scan(&categories).Error
	for _, row := range categories {
		ratio := 0.0
		if rows.TotalAmount > 0 {
			ratio = float64(row.TotalAmount) / float64(rows.TotalAmount)
		}
		rows.CategoryItems = append(rows.CategoryItems, usecase.CategorySummaryItem{CategoryID: row.CategoryID, CategoryName: row.CategoryName, Color: row.Color, TotalAmount: row.TotalAmount, TransactionCount: row.Count, Ratio: ratio})
	}

	var trend []struct {
		Label  string
		Amount int64
	}
	_ = baseQuery().Select(fmt.Sprintf("billing_month as label, coalesce(sum(%s), 0) as amount", amountColumn)).Group("billing_month").Order("label asc").Scan(&trend).Error
	for _, row := range trend {
		rows.Trend = append(rows.Trend, usecase.ChartPoint{Label: row.Label, Amount: row.Amount})
	}

	var frequent []struct {
		MerchantName string
		TotalAmount  int64
		Count        int64
	}
	maxAmount := f.MaxAmount
	if maxAmount <= 0 {
		maxAmount = 1000
	}
	_ = baseQuery().Where(fmt.Sprintf("coalesce(%s, 0) <= ?", amountColumn), maxAmount).
		Select(fmt.Sprintf("merchant_name as merchant_name, coalesce(sum(%s), 0) as total_amount, count(*) as count", amountColumn)).
		Group("merchant_name").Having("count(*) >= 3").Order("count desc").Limit(20).Scan(&frequent).Error
	for _, row := range frequent {
		avg := int64(0)
		if row.Count > 0 {
			avg = row.TotalAmount / row.Count
		}
		rows.SmallFrequent = append(rows.SmallFrequent, usecase.SmallFrequentCandidate{MerchantName: row.MerchantName, TotalAmount: row.TotalAmount, TransactionCount: row.Count, AverageAmount: avg})
	}

	return rows, nil
}

func (r transactionRepository) ApplyRule(ctx context.Context, rule domain.CategoryRule, overwrite bool) (int, int, error) {
	var models []TransactionModel
	query := r.db.WithContext(ctx).Model(&TransactionModel{})
	if !overwrite {
		query = query.Where("category_id is null")
	}
	if err := query.Find(&models).Error; err != nil {
		return 0, 0, err
	}
	matched := 0
	updated := 0
	for _, m := range models {
		if matchRule(rule, m.MerchantName) {
			matched++
			res := r.db.WithContext(ctx).Model(&TransactionModel{}).Where("id = ?", m.ID).Update("category_id", rule.CategoryID)
			if res.Error != nil {
				return matched, updated, res.Error
			}
			if res.RowsAffected > 0 {
				updated++
			}
		}
	}
	return matched, updated, nil
}

func (r transactionRepository) ListClassificationCandidates(ctx context.Context, limit int) ([]usecase.ClassificationCandidate, error) {
	var rows []struct {
		MerchantName string
		Count        int64
	}
	err := r.db.WithContext(ctx).Model(&TransactionModel{}).
		Select("merchant_name as merchant_name, count(*) as count").
		Where("category_id is null").
		Group("merchant_name").
		Order("count desc, merchant_name asc").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]usecase.ClassificationCandidate, 0, len(rows))
	for _, row := range rows {
		items = append(items, usecase.ClassificationCandidate{MerchantName: row.MerchantName, TransactionCount: row.Count})
	}
	return items, nil
}

func applyTransactionFilter(query *gorm.DB, f usecase.TransactionFilter) *gorm.DB {
	if f.BillingMonth != "" {
		query = query.Where("billing_month = ?", f.BillingMonth)
	}
	if f.UsageDateFrom != "" {
		query = query.Where("date(usage_date) >= ?", f.UsageDateFrom)
	}
	if f.UsageDateTo != "" {
		query = query.Where("date(usage_date) <= ?", f.UsageDateTo)
	}
	if f.MerchantName != "" {
		query = query.Where("merchant_name like ?", "%"+f.MerchantName+"%")
	}
	if f.CategoryID != "" {
		query = query.Where("category_id = ?", f.CategoryID)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		query = query.Where("merchant_name like ? or memo like ?", kw, kw)
	}
	if f.AmountMin != nil {
		query = query.Where("coalesce(billed_amount, usage_amount) >= ?", *f.AmountMin)
	}
	if f.AmountMax != nil {
		query = query.Where("coalesce(billed_amount, usage_amount) <= ?", *f.AmountMax)
	}
	if !f.IncludeExcluded {
		query = query.Where("excluded_from_analytics = ?", false)
	}
	return query
}

func mapSort(sort string) string {
	switch sort {
	case "merchantName":
		return "merchant_name"
	case "billingMonth":
		return "billing_month"
	case "usageAmount":
		return "usage_amount"
	case "billedAmount":
		return "billed_amount"
	default:
		return "usage_date"
	}
}

func matchRule(rule domain.CategoryRule, merchant string) bool {
	switch rule.MatchType {
	case "contains":
		return strings.Contains(merchant, rule.Pattern)
	case "startsWith":
		return strings.HasPrefix(merchant, rule.Pattern)
	case "equals":
		return merchant == rule.Pattern
	case "regex":
		re, err := regexp.Compile(rule.Pattern)
		return err == nil && re.MatchString(merchant)
	default:
		return false
	}
}
