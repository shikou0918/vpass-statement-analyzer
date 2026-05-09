package usecase

import (
	"context"
	"regexp"
	"strings"

	"vpass-statement-analyzer/backend/internal/domain"
)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (a *App) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return a.repos.Categories().List(ctx)
}

func (a *App) CreateCategory(ctx context.Context, in CategoryInput) (domain.Category, error) {
	if err := validateCategory(in); err != nil {
		return domain.Category{}, err
	}
	return a.repos.Categories().Create(ctx, domain.Category{Name: strings.TrimSpace(in.Name), Color: in.Color})
}

func (a *App) UpdateCategory(ctx context.Context, id int64, in CategoryInput) (*domain.Category, error) {
	if err := validateCategory(in); err != nil {
		return nil, err
	}
	return a.repos.Categories().Update(ctx, id, in)
}

func (a *App) DeleteCategory(ctx context.Context, id int64) error {
	return a.repos.Categories().Delete(ctx, id)
}

func validateCategory(in CategoryInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return BadRequest("カテゴリ名は必須です", map[string]any{"field": "name"})
	}
	if !colorPattern.MatchString(in.Color) {
		return BadRequest("色は #RRGGBB 形式で指定してください", map[string]any{"field": "color"})
	}
	return nil
}

func (a *App) ListCategoryRules(ctx context.Context) ([]domain.CategoryRule, error) {
	return a.repos.CategoryRules().List(ctx)
}

func (a *App) CreateCategoryRule(ctx context.Context, in CategoryRuleInput) (domain.CategoryRule, error) {
	if err := validateRule(in); err != nil {
		return domain.CategoryRule{}, err
	}
	return a.repos.CategoryRules().Create(ctx, domain.CategoryRule{MatchType: in.MatchType, Pattern: strings.TrimSpace(in.Pattern), CategoryID: in.CategoryID, Priority: in.Priority})
}

func (a *App) UpdateCategoryRule(ctx context.Context, id int64, in CategoryRuleInput) (*domain.CategoryRule, error) {
	if err := validateRule(in); err != nil {
		return nil, err
	}
	in.Pattern = strings.TrimSpace(in.Pattern)
	return a.repos.CategoryRules().Update(ctx, id, in)
}

func (a *App) DeleteCategoryRule(ctx context.Context, id int64) error {
	return a.repos.CategoryRules().Delete(ctx, id)
}

func (a *App) ApplyCategoryRules(ctx context.Context, overwrite bool) (matched, updated, unchanged, uncategorized int, err error) {
	rules, err := a.repos.CategoryRules().List(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for _, rule := range rules {
		m, u, err := a.repos.Transactions().ApplyRule(ctx, rule, overwrite)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		matched += m
		updated += u
	}
	if matched > updated {
		unchanged = matched - updated
	}
	return matched, updated, unchanged, 0, nil
}

func (a *App) ListClassificationCandidates(ctx context.Context, limit int) ([]ClassificationCandidate, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return a.repos.Transactions().ListClassificationCandidates(ctx, limit)
}

func validateRule(in CategoryRuleInput) error {
	switch in.MatchType {
	case "contains", "startsWith", "equals", "regex":
	default:
		return BadRequest("matchType が不正です", map[string]any{"field": "matchType"})
	}
	if strings.TrimSpace(in.Pattern) == "" {
		return BadRequest("pattern は必須です", map[string]any{"field": "pattern"})
	}
	if in.MatchType == "regex" {
		if _, err := regexp.Compile(in.Pattern); err != nil {
			return BadRequest("pattern が正規表現として不正です", map[string]any{"field": "pattern"})
		}
	}
	if in.CategoryID <= 0 {
		return BadRequest("categoryId は必須です", map[string]any{"field": "categoryId"})
	}
	return nil
}
