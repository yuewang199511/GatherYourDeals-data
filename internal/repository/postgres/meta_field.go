package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gatheryourdeals/data/internal/model"
)

const metaColumns = "field_name, description, field_type, native"

// MetaFieldRepo implements repository.MetaFieldRepository backed by PostgreSQL.
type MetaFieldRepo struct {
	db *DB
}

// NewMetaFieldRepo creates a new PostgreSQL-backed meta field repository.
func NewMetaFieldRepo(db *DB) *MetaFieldRepo {
	return &MetaFieldRepo{db: db}
}

func (r *MetaFieldRepo) CreateField(ctx context.Context, field *model.MetaField) error {
	query := `INSERT INTO meta_fields (` + metaColumns + `) VALUES ($1, $2, $3, $4)`
	_, err := r.db.conn.ExecContext(ctx, query,
		field.FieldName, field.Description, field.FieldType, field.Native)
	if err != nil {
		return fmt.Errorf("create meta field: %w", err)
	}
	return nil
}

func (r *MetaFieldRepo) GetField(ctx context.Context, fieldName string) (*model.MetaField, error) {
	query := `SELECT ` + metaColumns + ` FROM meta_fields WHERE field_name = $1`
	row := r.db.conn.QueryRowContext(ctx, query, fieldName)

	var f model.MetaField
	err := row.Scan(&f.FieldName, &f.Description, &f.FieldType, &f.Native)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get meta field: %w", err)
	}
	return &f, nil
}

func (r *MetaFieldRepo) ListFields(ctx context.Context, params model.PaginationParams) (*model.Page[*model.MetaField], error) {
	// Count total fields.
	var total int
	if err := r.db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM meta_fields`).Scan(&total); err != nil {
		return nil, fmt.Errorf("count meta fields: %w", err)
	}

	page := &model.Page[*model.MetaField]{
		Data:   []*model.MetaField{},
		Total:  total,
		Offset: params.Offset,
		Limit:  params.Limit,
	}
	if total > 0 {
		page.TotalPages = (total + params.Limit - 1) / params.Limit
	}

	if total == 0 || params.Offset >= total {
		page.Data = []*model.MetaField{}
		return page, nil
	}

	// Fetch paginated data. SortBy and SortOrder are validated by the handler.
	query := fmt.Sprintf(
		`SELECT `+metaColumns+` FROM meta_fields ORDER BY %s %s LIMIT $1 OFFSET $2`,
		params.SortBy, params.SortOrder,
	)
	rows, err := r.db.conn.QueryContext(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("list meta fields: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var fields []*model.MetaField
	for rows.Next() {
		var f model.MetaField
		if err := rows.Scan(&f.FieldName, &f.Description, &f.FieldType, &f.Native); err != nil {
			return nil, fmt.Errorf("scan meta field: %w", err)
		}
		fields = append(fields, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = []*model.MetaField{}
	}
	page.Data = fields
	return page, nil
}

func (r *MetaFieldRepo) UpdateDescription(ctx context.Context, fieldName string, description string) error {
	result, err := r.db.conn.ExecContext(ctx,
		`UPDATE meta_fields SET description = $1 WHERE field_name = $2`,
		description, fieldName)
	if err != nil {
		return fmt.Errorf("update meta field description: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %q", model.ErrFieldNotFound, fieldName)
	}
	return nil
}
