package database

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vpass-statement-analyzer/backend/internal/domain"
)

type creditCardRepository struct {
	db *gorm.DB
}

func (r creditCardRepository) List(ctx context.Context) ([]domain.CreditCard, error) {
	var models []CreditCardModel
	if err := r.db.WithContext(ctx).Order("display_name asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.CreditCard, 0, len(models))
	for _, m := range models {
		items = append(items, creditCardToDomain(m))
	}
	return items, nil
}

func (r creditCardRepository) FindOrCreateByDisplayName(ctx context.Context, displayName string) (domain.CreditCard, error) {
	name := strings.TrimSpace(displayName)
	model := CreditCardModel{DisplayName: name}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "display_name"}}, DoNothing: true}).Create(&model).Error
	if err != nil {
		return domain.CreditCard{}, err
	}
	if model.ID == 0 {
		err = r.db.WithContext(ctx).Where("display_name = ?", name).First(&model).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.CreditCard{}, nil
	}
	if err != nil {
		return domain.CreditCard{}, err
	}
	return creditCardToDomain(model), nil
}
