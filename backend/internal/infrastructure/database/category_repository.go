package database

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"vpass-statement-analyzer/backend/internal/domain"
	"vpass-statement-analyzer/backend/internal/usecase"
)

type categoryRepository struct {
	db *gorm.DB
}

func (r categoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	var models []CategoryModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Category, 0, len(models))
	for _, m := range models {
		items = append(items, categoryToDomain(m))
	}
	return items, nil
}

func (r categoryRepository) Create(ctx context.Context, c domain.Category) (domain.Category, error) {
	model := CategoryModel{Name: c.Name, Color: c.Color, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.Category{}, err
	}
	return categoryToDomain(model), nil
}

func (r categoryRepository) Update(ctx context.Context, id int64, in usecase.CategoryInput) (*domain.Category, error) {
	res := r.db.WithContext(ctx).Model(&CategoryModel{}).Where("id = ?", id).Updates(map[string]any{"name": in.Name, "color": in.Color, "updated_at": time.Now()})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, usecase.NotFound("カテゴリが見つかりません")
	}
	var model CategoryModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	item := categoryToDomain(model)
	return &item, nil
}

func (r categoryRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&TransactionModel{}).Where("category_id = ?", id).Update("category_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("category_id = ?", id).Delete(&CategoryRuleModel{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&CategoryModel{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return usecase.NotFound("カテゴリが見つかりません")
		}
		return nil
	})
}

type categoryRuleRepository struct {
	db *gorm.DB
}

func (r categoryRuleRepository) List(ctx context.Context) ([]domain.CategoryRule, error) {
	var models []CategoryRuleModel
	if err := r.db.WithContext(ctx).Order("priority asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.CategoryRule, 0, len(models))
	for _, m := range models {
		items = append(items, ruleToDomain(m))
	}
	return items, nil
}

func (r categoryRuleRepository) Create(ctx context.Context, rule domain.CategoryRule) (domain.CategoryRule, error) {
	if err := ensureCategory(ctx, r.db, rule.CategoryID); err != nil {
		return domain.CategoryRule{}, err
	}
	if err := ensureUniqueCategoryRule(ctx, r.db, 0, rule.MatchType, rule.Pattern, rule.CategoryID); err != nil {
		return domain.CategoryRule{}, err
	}
	model := CategoryRuleModel{MatchType: rule.MatchType, Pattern: rule.Pattern, CategoryID: rule.CategoryID, Priority: rule.Priority, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.CategoryRule{}, err
	}
	return ruleToDomain(model), nil
}

func (r categoryRuleRepository) Update(ctx context.Context, id int64, in usecase.CategoryRuleInput) (*domain.CategoryRule, error) {
	if err := ensureCategory(ctx, r.db, in.CategoryID); err != nil {
		return nil, err
	}
	if err := ensureUniqueCategoryRule(ctx, r.db, id, in.MatchType, in.Pattern, in.CategoryID); err != nil {
		return nil, err
	}
	res := r.db.WithContext(ctx).Model(&CategoryRuleModel{}).Where("id = ?", id).Updates(map[string]any{"match_type": in.MatchType, "pattern": in.Pattern, "category_id": in.CategoryID, "priority": in.Priority, "updated_at": time.Now()})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, usecase.NotFound("分類ルールが見つかりません")
	}
	var model CategoryRuleModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	item := ruleToDomain(model)
	return &item, nil
}

func (r categoryRuleRepository) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&CategoryRuleModel{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return usecase.NotFound("分類ルールが見つかりません")
	}
	return nil
}

func ensureCategory(ctx context.Context, db *gorm.DB, id int64) error {
	var model CategoryModel
	err := db.WithContext(ctx).First(&model, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return usecase.BadRequest("categoryId が存在しません", map[string]any{"field": "categoryId"})
	}
	return err
}

func ensureUniqueCategoryRule(ctx context.Context, db *gorm.DB, excludeID int64, matchType string, pattern string, categoryID int64) error {
	query := db.WithContext(ctx).
		Where("match_type = ? AND pattern = ? AND category_id = ?", matchType, pattern, categoryID)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var model CategoryRuleModel
	err := query.First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return usecase.Conflict("同じ分類ルールが既に存在します", map[string]any{
		"matchType":  matchType,
		"pattern":    pattern,
		"categoryId": categoryID,
	})
}
