package usecase

import (
	"context"
	"time"
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
	previousAmount, err := a.previousMonthAmount(ctx, f)
	if err != nil {
		return MonthlySummaryResponse{}, err
	}
	return MonthlySummaryResponse{
		Month:            f.Month,
		TotalAmount:      rows.TotalAmount,
		PreviousAmount:   previousAmount,
		DiffAmount:       rows.TotalAmount - previousAmount,
		TransactionCount: rows.TransactionCount,
		DailyTrend:       rows.Daily,
	}, nil
}

func (a *App) previousMonthAmount(ctx context.Context, f SummaryFilter) (int64, error) {
	month, err := time.Parse("2006-01", f.Month)
	if err != nil {
		return 0, BadRequest("month は YYYY-MM 形式で指定してください", map[string]any{"field": "month"})
	}
	previousFilter := f
	previousFilter.Month = month.AddDate(0, -1, 0).Format("2006-01")
	rows, err := a.repos.Transactions().Summary(ctx, previousFilter)
	if err != nil {
		return 0, err
	}
	return rows.TotalAmount, nil
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
