package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gatheryourdeals/data/internal/model"
)

const receiptColumns = "id, product_name, purchase_date, price, amount, store_name, latitude, longitude, extras, upload_time, user_id"

// ReceiptRepo implements repository.ReceiptRepository backed by PostgreSQL.
type ReceiptRepo struct {
	db   *DB
	meta *MetaFieldRepo
}

// NewReceiptRepo creates a new PostgreSQL-backed receipt repository.
// It takes a MetaFieldRepo to validate extra fields on insert.
func NewReceiptRepo(db *DB, meta *MetaFieldRepo) *ReceiptRepo {
	return &ReceiptRepo{db: db, meta: meta}
}

func (r *ReceiptRepo) CreateReceipt(ctx context.Context, receipt *model.Receipt) error {
	if err := r.validateExtras(ctx, receipt.Extras); err != nil {
		return err
	}

	extrasJSON, err := json.Marshal(receipt.Extras)
	if err != nil {
		return fmt.Errorf("marshal extras: %w", err)
	}
	if receipt.Extras == nil {
		extrasJSON = []byte("{}")
	}

	receipt.UploadTime = time.Now().Unix()

	query := `INSERT INTO receipts (` + receiptColumns + `) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = r.db.conn.ExecContext(ctx, query,
		receipt.ID,
		receipt.ProductName,
		receipt.PurchaseDate,
		receipt.Price,
		receipt.Amount,
		receipt.StoreName,
		receipt.Latitude,
		receipt.Longitude,
		string(extrasJSON),
		receipt.UploadTime,
		receipt.UserID,
	)
	if err != nil {
		return fmt.Errorf("create receipt: %w", err)
	}
	return nil
}

func (r *ReceiptRepo) GetReceiptByID(ctx context.Context, id string) (*model.Receipt, error) {
	query := `SELECT ` + receiptColumns + ` FROM receipts WHERE id = $1`
	row := r.db.conn.QueryRowContext(ctx, query, id)
	return r.scanReceipt(row)
}

func (r *ReceiptRepo) ListReceiptsByUser(ctx context.Context, userID string, params model.PaginationParams) (*model.Page[*model.Receipt], error) {
	// Count total matching records.
	var total int
	if err := r.db.conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM receipts WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count receipts: %w", err)
	}

	page := &model.Page[*model.Receipt]{
		Data:   []*model.Receipt{},
		Total:  total,
		Offset: params.Offset,
		Limit:  params.Limit,
	}
	if total > 0 {
		page.TotalPages = (total + params.Limit - 1) / params.Limit
	}

	if total == 0 || params.Offset >= total {
		page.Data = []*model.Receipt{}
		return page, nil
	}

	// Fetch paginated data. SortBy and SortOrder are validated by the handler.
	query := fmt.Sprintf(
		`SELECT `+receiptColumns+` FROM receipts WHERE user_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3`,
		params.SortBy, params.SortOrder,
	)
	rows, err := r.db.conn.QueryContext(ctx, query, userID, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var receipts []*model.Receipt
	for rows.Next() {
		rec, err := r.scanReceiptRow(rows)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if receipts == nil {
		receipts = []*model.Receipt{}
	}
	page.Data = receipts
	return page, nil
}

func (r *ReceiptRepo) DeleteReceipt(ctx context.Context, id string) error {
	_, err := r.db.conn.ExecContext(ctx, `DELETE FROM receipts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete receipt: %w", err)
	}
	return nil
}

// validateExtras checks that every key in the extras map is registered in the meta table.
func (r *ReceiptRepo) validateExtras(ctx context.Context, extras map[string]interface{}) error {
	if len(extras) == 0 {
		return nil
	}
	for key := range extras {
		field, err := r.meta.GetField(ctx, key)
		if err != nil {
			return fmt.Errorf("validate extras: %w", err)
		}
		if field == nil {
			slog.Warn("receipt rejected: unregistered field in extras", "field", key)
			return fmt.Errorf("%w: %q", model.ErrFieldNotRegistered, key)
		}
		if field.Native {
			slog.Warn("receipt rejected: native field used in extras", "field", key)
			return fmt.Errorf("%w: %q is a native field and cannot be used in extras", model.ErrFieldNotRegistered, key)
		}
	}
	return nil
}

// scanReceipt scans a single receipt from a QueryRow result.
func (r *ReceiptRepo) scanReceipt(row *sql.Row) (*model.Receipt, error) {
	var rec model.Receipt
	var extrasStr string
	err := row.Scan(
		&rec.ID, &rec.ProductName, &rec.PurchaseDate,
		&rec.Price, &rec.Amount, &rec.StoreName,
		&rec.Latitude, &rec.Longitude, &extrasStr,
		&rec.UploadTime, &rec.UserID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan receipt: %w", err)
	}
	if err := json.Unmarshal([]byte(extrasStr), &rec.Extras); err != nil {
		return nil, fmt.Errorf("unmarshal extras: %w", err)
	}
	return &rec, nil
}

// scanReceiptRow scans a single receipt from a multi-row result set.
func (r *ReceiptRepo) scanReceiptRow(rows *sql.Rows) (*model.Receipt, error) {
	var rec model.Receipt
	var extrasStr string
	err := rows.Scan(
		&rec.ID, &rec.ProductName, &rec.PurchaseDate,
		&rec.Price, &rec.Amount, &rec.StoreName,
		&rec.Latitude, &rec.Longitude, &extrasStr,
		&rec.UploadTime, &rec.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("scan receipt row: %w", err)
	}
	if err := json.Unmarshal([]byte(extrasStr), &rec.Extras); err != nil {
		return nil, fmt.Errorf("unmarshal extras: %w", err)
	}
	return &rec, nil
}
