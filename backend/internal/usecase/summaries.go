package usecase

import (
	"context"

	"vpass-statement-analyzer/backend/internal/domain"
)

type MonthlySummaryResponse struct {
	Month            string       `json:"month"`
	TotalAmount      int64        `json:"totalAmount"`
	PreviousAmount   int64        `json:"previousAmount"`
	DiffAmount       int64        `json:"diffAmount"`
	TransactionCount int64        `json:"transactionCount"`
	DailyTrend       []ChartPoint `json:"dailyTrend"`
}

type RankingSummaryResponse struct {
	Items []RankingItem `json:"items"`
}

type CategorySummaryResponse struct {
	Items []CategorySummaryItem `json:"items"`
}

type TrendResponse struct {
	Items []ChartPoint `json:"items"`
}

func (a *App) MonthlySummary(ctx context.Context, f SummaryFilter) (MonthlySummaryResponse, error) {
	if f.Month == "" {
		return MonthlySummaryResponse{}, BadRequest("month は必須です", map[string]any{"field": "month"})
	}
	rows, err := a.repos.Transactions().Summary(ctx, f)
	if err != nil {
		return MonthlySummaryResponse{}, err
	}
	return MonthlySummaryResponse{
		Month:            f.Month,
		TotalAmount:      rows.TotalAmount,
		PreviousAmount:   rows.PreviousAmount,
		DiffAmount:       rows.TotalAmount - rows.PreviousAmount,
		TransactionCount: rows.TransactionCount,
		DailyTrend:       rows.Daily,
	}, nil
}

func (a *App) MerchantSummary(ctx context.Context, f SummaryFilter) (RankingSummaryResponse, error) {
	rows, err := a.repos.Transactions().Summary(ctx, f)
	return RankingSummaryResponse{Items: rows.Ranking}, err
}

func (a *App) CategorySummary(ctx context.Context, f SummaryFilter) (CategorySummaryResponse, error) {
	rows, err := a.repos.Transactions().Summary(ctx, f)
	return CategorySummaryResponse{Items: rows.CategoryItems}, err
}

func (a *App) Trend(ctx context.Context, f SummaryFilter) (TrendResponse, error) {
	rows, err := a.repos.Transactions().Summary(ctx, f)
	return TrendResponse{Items: rows.Trend}, err
}

func (a *App) RecurringCandidates(ctx context.Context, f SummaryFilter) ([]RecurringCandidate, error) {
	rows, err := a.repos.Transactions().Summary(ctx, f)
	return rows.Recurring, err
}

func (a *App) SmallFrequent(ctx context.Context, f SummaryFilter) ([]SmallFrequentCandidate, error) {
	rows, err := a.repos.Transactions().Summary(ctx, f)
	return rows.SmallFrequent, err
}

func (a *App) GetSettings(ctx context.Context) (domain.AppSettings, error) {
	return a.repos.Settings().Get(ctx)
}

func (a *App) UpdateSettings(ctx context.Context, in domain.AppSettings) (domain.AppSettings, error) {
	if in.DefaultBasisDate != "billingMonth" && in.DefaultBasisDate != "usageDate" {
		return domain.AppSettings{}, BadRequest("defaultBasisDate が不正です", map[string]any{"field": "defaultBasisDate"})
	}
	if in.DefaultBasisAmount != "billedAmount" && in.DefaultBasisAmount != "usageAmount" {
		return domain.AppSettings{}, BadRequest("defaultBasisAmount が不正です", map[string]any{"field": "defaultBasisAmount"})
	}
	return a.repos.Settings().Update(ctx, in)
}
