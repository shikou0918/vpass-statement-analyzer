package usecase

import (
	"context"
	"strconv"

	"vpass-statement-analyzer/backend/internal/domain"
)

func (a *App) ListTransactions(ctx context.Context, f TransactionFilter) ([]domain.Transaction, Pagination, error) {
	f.Page, f.PageSize = normalizePage(f.Page, f.PageSize)
	if f.Order != "" && f.Order != "asc" && f.Order != "desc" {
		return nil, Pagination{}, BadRequest("order は asc または desc を指定してください", nil)
	}
	items, total, err := a.repos.Transactions().List(ctx, f)
	if err != nil {
		return nil, Pagination{}, err
	}
	return items, pagination(f.Page, f.PageSize, total), nil
}

func (a *App) GetTransaction(ctx context.Context, id int64) (*domain.Transaction, error) {
	return a.repos.Transactions().FindByID(ctx, id)
}

func (a *App) UpdateTransaction(ctx context.Context, id int64, in UpdateTransactionInput) (*domain.Transaction, error) {
	return a.repos.Transactions().UpdateEditable(ctx, id, in)
}

func ParseOptionalID(s *string) (*int64, bool, error) {
	if s == nil {
		return nil, false, nil
	}
	if *s == "" {
		return nil, true, nil
	}
	id, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil, true, BadRequest("ID形式が不正です", map[string]any{"id": *s})
	}
	return &id, true, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func pagination(page, pageSize int, total int64) Pagination {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	return Pagination{Page: page, PageSize: pageSize, TotalItems: total, TotalPages: totalPages}
}
