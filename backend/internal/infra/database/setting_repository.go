package database

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vpass-statement-analyzer/backend/internal/domain"
)

type settingRepository struct {
	db *gorm.DB
}

func (r settingRepository) Get(ctx context.Context) (domain.AppSettings, error) {
	settings := domain.AppSettings{DefaultBasisDate: "billingMonth", DefaultBasisAmount: "billedAmount"}
	var rows []SettingModel
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return settings, err
	}
	for _, row := range rows {
		switch row.Key {
		case "defaultBasisDate":
			settings.DefaultBasisDate = row.Value
		case "defaultBasisAmount":
			settings.DefaultBasisAmount = row.Value
		}
	}
	return settings, nil
}

func (r settingRepository) Update(ctx context.Context, s domain.AppSettings) (domain.AppSettings, error) {
	rows := []SettingModel{
		{Key: "defaultBasisDate", Value: s.DefaultBasisDate, UpdatedAt: time.Now()},
		{Key: "defaultBasisAmount", Value: s.DefaultBasisAmount, UpdatedAt: time.Now()},
	}
	for _, row := range rows {
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return domain.AppSettings{}, err
		}
	}
	return r.Get(ctx)
}
