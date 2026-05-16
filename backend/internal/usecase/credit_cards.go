package usecase

import (
	"context"

	"vpass-statement-analyzer/backend/internal/domain"
)

func (a *App) ListCreditCards(ctx context.Context) ([]domain.CreditCard, error) {
	return a.repos.CreditCards().List(ctx)
}
