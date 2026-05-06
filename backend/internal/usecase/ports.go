package usecase

import (
	"context"

	"vpass-statement-analyzer/backend/internal/domain"
)

type TxRepositories interface {
	Imports() ImportRepository
	Transactions() TransactionRepository
	Categories() CategoryRepository
	CategoryRules() CategoryRuleRepository
	Settings() SettingRepository
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repos TxRepositories) error) error
}

type Repositories interface {
	TxRepositories
}

type ImportRepository interface {
	CreateImport(ctx context.Context, file domain.ImportFile, mappings []domain.ImportMapping, errs []domain.ImportError) (domain.ImportFile, error)
	FindByHash(ctx context.Context, hash string) (*domain.ImportFile, error)
	FindByID(ctx context.Context, id int64) (*domain.ImportFile, error)
	List(ctx context.Context, page, pageSize int) ([]domain.ImportFile, int64, error)
	Delete(ctx context.Context, id int64) error
}

type TransactionRepository interface {
	CreateMany(ctx context.Context, txs []domain.Transaction) (created int, skipped int, err error)
	List(ctx context.Context, f TransactionFilter) ([]domain.Transaction, int64, error)
	FindByID(ctx context.Context, id int64) (*domain.Transaction, error)
	UpdateEditable(ctx context.Context, id int64, in UpdateTransactionInput) (*domain.Transaction, error)
	DeleteByImportID(ctx context.Context, importID int64) error
	Summary(ctx context.Context, f SummaryFilter) (SummaryRows, error)
	ApplyRule(ctx context.Context, rule domain.CategoryRule, overwrite bool) (matched int, updated int, err error)
}

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	Create(ctx context.Context, c domain.Category) (domain.Category, error)
	Update(ctx context.Context, id int64, in CategoryInput) (*domain.Category, error)
	Delete(ctx context.Context, id int64) error
}

type CategoryRuleRepository interface {
	List(ctx context.Context) ([]domain.CategoryRule, error)
	Create(ctx context.Context, r domain.CategoryRule) (domain.CategoryRule, error)
	Update(ctx context.Context, id int64, in CategoryRuleInput) (*domain.CategoryRule, error)
	Delete(ctx context.Context, id int64) error
}

type SettingRepository interface {
	Get(ctx context.Context) (domain.AppSettings, error)
	Update(ctx context.Context, s domain.AppSettings) (domain.AppSettings, error)
}
