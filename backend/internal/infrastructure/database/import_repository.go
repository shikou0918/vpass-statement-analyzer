package database

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"vpass-statement-analyzer/backend/internal/domain"
	"vpass-statement-analyzer/backend/internal/usecase"
)

type importRepository struct {
	db *gorm.DB
}

func (r importRepository) CreateImport(ctx context.Context, file domain.ImportFile, mappings []domain.ImportMapping, errs []domain.ImportError) (domain.ImportFile, error) {
	model := ImportFileModel{FileName: file.FileName, FileHash: file.FileHash, CreditCardID: file.CreditCardID, DetectedFormat: file.DetectedFormat, HasHeader: file.HasHeader, RowCount: file.RowCount, ImportedAt: time.Now()}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.ImportFile{}, err
	}
	for _, m := range mappings {
		row := ImportMappingModel{SourceFileID: model.ID, SourceColumnName: m.SourceColumnName, SourceColumnIndex: m.SourceColumnIndex, TargetField: m.TargetField, Confidence: m.Confidence, CreatedAt: time.Now()}
		if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
			return domain.ImportFile{}, err
		}
	}
	for _, e := range errs {
		raw, _ := json.Marshal(e.RawColumns)
		row := ImportErrorModel{SourceFileID: model.ID, RowNumber: e.RowNumber, ErrorType: e.ErrorType, Message: e.Message, RawColumns: string(raw), CreatedAt: time.Now()}
		if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
			return domain.ImportFile{}, err
		}
	}
	return importFileToDomain(model), nil
}

func (r importRepository) FindByHash(ctx context.Context, hash string) (*domain.ImportFile, error) {
	var model ImportFileModel
	err := r.db.WithContext(ctx).Where("file_hash = ?", hash).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := importFileToDomain(model)
	return &item, nil
}

func (r importRepository) FindByID(ctx context.Context, id int64) (*domain.ImportFile, error) {
	var model ImportFileModel
	err := r.db.WithContext(ctx).First(&model, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, usecase.NotFound("インポートが見つかりません")
	}
	if err != nil {
		return nil, err
	}
	item := importFileToDomain(model)
	return &item, nil
}

func (r importRepository) List(ctx context.Context, page, pageSize int) ([]domain.ImportFile, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&ImportFileModel{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []ImportFileModel
	if err := query.Order("imported_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.ImportFile, 0, len(models))
	for _, m := range models {
		items = append(items, importFileToDomain(m))
	}
	return items, total, nil
}

func (r importRepository) UpdateCreditCard(ctx context.Context, id int64, creditCardID *int64) (*domain.ImportFile, error) {
	res := r.db.WithContext(ctx).Model(&ImportFileModel{}).Where("id = ?", id).Update("credit_card_id", creditCardID)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, usecase.NotFound("インポートが見つかりません")
	}
	return r.FindByID(ctx, id)
}

func (r importRepository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Where("source_file_id = ?", id).Delete(&ImportMappingModel{}).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Where("source_file_id = ?", id).Delete(&ImportErrorModel{}).Error; err != nil {
		return err
	}
	res := r.db.WithContext(ctx).Delete(&ImportFileModel{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return usecase.NotFound("インポートが見つかりません")
	}
	return nil
}
