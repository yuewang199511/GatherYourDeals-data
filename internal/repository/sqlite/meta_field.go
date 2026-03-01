package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gatheryourdeals/data/internal/model"
)

const metaColumns = "field_name, description, field_type, native"

// MetaFieldRepo implements repository.MetaFieldRepository backed by SQLite.
type MetaFieldRepo struct {
	db *DB
}

// NewMetaFieldRepo creates a new SQLite-backed meta field repository.
func NewMetaFieldRepo(db *DB) *MetaFieldRepo {
	return &MetaFieldRepo{db: db}
}

func (r *MetaFieldRepo) CreateField(ctx context.Context, field *model.MetaField) error {
	query := `INSERT INTO meta_fields (` + metaColumns + `) VALUES (?, ?, ?, ?)`
	native := 0
	if field.Native {
		native = 1
	}
	_, err := r.db.conn.ExecContext(ctx, query,
		field.FieldName, field.Description, field.FieldType, native)
	if err != nil {
		return fmt.Errorf("create meta field: %w", err)
	}
	return nil
}

func (r *MetaFieldRepo) GetField(ctx context.Context, fieldName string) (*model.MetaField, error) {
	query := `SELECT ` + metaColumns + ` FROM meta_fields WHERE field_name = ?`
	row := r.db.conn.QueryRowContext(ctx, query, fieldName)

	var f model.MetaField
	var native int
	err := row.Scan(&f.FieldName, &f.Description, &f.FieldType, &native)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get meta field: %w", err)
	}
	f.Native = native == 1
	return &f, nil
}

func (r *MetaFieldRepo) ListFields(ctx context.Context) ([]*model.MetaField, error) {
	rows, err := r.db.conn.QueryContext(ctx, `SELECT `+metaColumns+` FROM meta_fields ORDER BY native DESC, field_name`)
	if err != nil {
		return nil, fmt.Errorf("list meta fields: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var fields []*model.MetaField
	for rows.Next() {
		var f model.MetaField
		var native int
		if err := rows.Scan(&f.FieldName, &f.Description, &f.FieldType, &native); err != nil {
			return nil, fmt.Errorf("scan meta field: %w", err)
		}
		f.Native = native == 1
		fields = append(fields, &f)
	}
	return fields, rows.Err()
}

func (r *MetaFieldRepo) UpdateDescription(ctx context.Context, fieldName string, description string) error {
	result, err := r.db.conn.ExecContext(ctx,
		`UPDATE meta_fields SET description = ? WHERE field_name = ?`,
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

