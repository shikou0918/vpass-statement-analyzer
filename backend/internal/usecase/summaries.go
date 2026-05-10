package usecase

import "context"

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
