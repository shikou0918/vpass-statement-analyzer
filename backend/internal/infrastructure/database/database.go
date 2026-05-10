package database

import (
	"context"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vpass-statement-analyzer/backend/internal/usecase"
)

func Open(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ImportFileModel{}, &ImportMappingModel{}, &ImportErrorModel{}, &TransactionModel{}, &CategoryModel{}, &CategoryRuleModel{})
}

type Repositories struct {
	db *gorm.DB
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{db: db}
}

func (r *Repositories) Imports() usecase.ImportRepository { return importRepository{db: r.db} }
func (r *Repositories) Transactions() usecase.TransactionRepository {
	return transactionRepository{db: r.db}
}
func (r *Repositories) Categories() usecase.CategoryRepository { return categoryRepository{db: r.db} }
func (r *Repositories) CategoryRules() usecase.CategoryRuleRepository {
	return categoryRuleRepository{db: r.db}
}

type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repos usecase.TxRepositories) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRepositories(tx))
	})
}
