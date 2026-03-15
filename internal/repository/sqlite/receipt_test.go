package sqlite_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

type receiptEnv struct {
	receipts *sqlite.ReceiptRepo
	meta     *sqlite.MetaFieldRepo
	users    *sqlite.UserRepo
	ctx      context.Context
}

func newReceiptEnv(t *testing.T) *receiptEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	meta := sqlite.NewMetaFieldRepo(db)
	return &receiptEnv{
		receipts: sqlite.NewReceiptRepo(db, meta),
		meta:     meta,
		users:    sqlite.NewUserRepo(db),
		ctx:      context.Background(),
	}
}

func (e *receiptEnv) seedUser(t *testing.T, id string) {
	t.Helper()
	err := e.users.CreateUser(e.ctx, &model.User{
		ID:           id,
		Username:     id,
		PasswordHash: "hash",
		Role:         model.RoleUser,
	})
	if err != nil {
		t.Fatalf("seedUser failed: %v", err)
	}
}

func (e *receiptEnv) sampleReceipt(id, userID string) *model.Receipt {
	return &model.Receipt{
		ID:           id,
		ProductName:  "Milk 2%",
		PurchaseDate: "2025.04.05",
		Price:        "5.49CAD",
		Amount:       "1",
		StoreName:    "Costco",
		UserID:       userID,
	}
}

// defaultReceiptParams returns pagination params suitable for use in tests that
// don't need to test pagination specifically (fetches all, default sort).
func defaultReceiptParams() model.PaginationParams {
	return model.PaginationParams{Offset: 0, Limit: 100, SortBy: "upload_time", SortOrder: "DESC"}
}

func TestReceipt_CreateAndGetByID(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	if rec.UploadTime == 0 {
		t.Error("expected UploadTime to be set")
	}

	got, err := env.receipts.GetReceiptByID(env.ctx, "r-1")
	if err != nil {
		t.Fatalf("GetReceiptByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected receipt, got nil")
	}
	if got.ProductName != "Milk 2%" {
		t.Errorf("expected ProductName 'Milk 2%%', got %q", got.ProductName)
	}
	if got.UserID != "user-1" {
		t.Errorf("expected UserID 'user-1', got %q", got.UserID)
	}
}

func TestReceipt_GetByID_NotFound(t *testing.T) {
	env := newReceiptEnv(t)

	got, err := env.receipts.GetReceiptByID(env.ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestReceipt_WithOptionalLatLon(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	lat := 49.2827
	lon := -123.1207
	rec := env.sampleReceipt("r-1", "user-1")
	rec.Latitude = &lat
	rec.Longitude = &lon

	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	got, err := env.receipts.GetReceiptByID(env.ctx, "r-1")
	if err != nil {
		t.Fatalf("GetReceiptByID failed: %v", err)
	}
	if got.Latitude == nil || *got.Latitude != 49.2827 {
		t.Errorf("expected latitude 49.2827, got %v", got.Latitude)
	}
	if got.Longitude == nil || *got.Longitude != -123.1207 {
		t.Errorf("expected longitude -123.1207, got %v", got.Longitude)
	}
}

func TestReceipt_WithNilLatLon(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	// Latitude and Longitude are nil by default.

	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	got, err := env.receipts.GetReceiptByID(env.ctx, "r-1")
	if err != nil {
		t.Fatalf("GetReceiptByID failed: %v", err)
	}
	if got.Latitude != nil {
		t.Errorf("expected nil latitude, got %v", got.Latitude)
	}
	if got.Longitude != nil {
		t.Errorf("expected nil longitude, got %v", got.Longitude)
	}
}

func TestReceipt_WithExtras(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	// Register a custom field first.
	if err := env.meta.CreateField(env.ctx, &model.MetaField{
		FieldName:   "brand",
		Description: "brand of the product",
		FieldType:   "string",
	}); err != nil {
		t.Fatalf("CreateField failed: %v", err)
	}

	rec := env.sampleReceipt("r-1", "user-1")
	rec.Extras = map[string]interface{}{"brand": "Kirkland"}

	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	got, err := env.receipts.GetReceiptByID(env.ctx, "r-1")
	if err != nil {
		t.Fatalf("GetReceiptByID failed: %v", err)
	}
	if got.Extras["brand"] != "Kirkland" {
		t.Errorf("expected extras brand 'Kirkland', got %v", got.Extras["brand"])
	}
}

func TestReceipt_ExtrasValidation_NativeFieldRejected(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	rec.Extras = map[string]interface{}{"productName": "duplicate"}

	err := env.receipts.CreateReceipt(env.ctx, rec)
	if err == nil {
		t.Fatal("expected error for native field in extras, got nil")
	}
}

func TestReceipt_ExtrasValidation_UnregisteredField(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	rec.Extras = map[string]interface{}{"unknownField": "value"}

	err := env.receipts.CreateReceipt(env.ctx, rec)
	if err == nil {
		t.Fatal("expected error for unregistered extra field, got nil")
	}
}

func TestReceipt_ListByUser(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")
	env.seedUser(t, "user-2")

	for _, id := range []string{"r-1", "r-2", "r-3"} {
		rec := env.sampleReceipt(id, "user-1")
		if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
			t.Fatalf("CreateReceipt %s failed: %v", id, err)
		}
	}

	// One receipt for user-2
	rec := env.sampleReceipt("r-4", "user-2")
	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	page, err := env.receipts.ListReceiptsByUser(env.ctx, "user-1", defaultReceiptParams())
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("expected total 3 for user-1, got %d", page.Total)
	}
	if len(page.Data) != 3 {
		t.Errorf("expected 3 receipts in data for user-1, got %d", len(page.Data))
	}

	page2, err := env.receipts.ListReceiptsByUser(env.ctx, "user-2", defaultReceiptParams())
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if page2.Total != 1 {
		t.Errorf("expected total 1 for user-2, got %d", page2.Total)
	}
}

func TestReceipt_ListByUser_Empty(t *testing.T) {
	env := newReceiptEnv(t)

	page, err := env.receipts.ListReceiptsByUser(env.ctx, "nobody", defaultReceiptParams())
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("expected total 0, got %d", page.Total)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(page.Data))
	}
	if page.Data == nil {
		t.Error("expected non-nil Data slice, got nil")
	}
	if page.TotalPages != 0 {
		t.Errorf("expected total_pages 0, got %d", page.TotalPages)
	}
}

func TestReceipt_Pagination_LimitOffset(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	// Create 5 receipts.
	for i := 1; i <= 5; i++ {
		rec := env.sampleReceipt(fmt.Sprintf("r-%d", i), "user-1")
		if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
			t.Fatalf("CreateReceipt r-%d failed: %v", i, err)
		}
	}

	params := model.PaginationParams{Offset: 2, Limit: 2, SortBy: "upload_time", SortOrder: "DESC"}
	page, err := env.receipts.ListReceiptsByUser(env.ctx, "user-1", params)
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if page.Total != 5 {
		t.Errorf("expected total 5, got %d", page.Total)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 receipts in page, got %d", len(page.Data))
	}
	if page.TotalPages != 3 {
		t.Errorf("expected total_pages 3, got %d", page.TotalPages)
	}
}

func TestReceipt_Pagination_OffsetBeyondTotal(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	params := model.PaginationParams{Offset: 100, Limit: 10, SortBy: "upload_time", SortOrder: "DESC"}
	page, err := env.receipts.ListReceiptsByUser(env.ctx, "user-1", params)
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("expected total 1, got %d", page.Total)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(page.Data))
	}
}

func TestReceipt_Pagination_TotalPagesExact(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	// Create exactly 4 receipts — divisible by limit=2.
	for i := 1; i <= 4; i++ {
		rec := env.sampleReceipt(fmt.Sprintf("r-%d", i), "user-1")
		if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
			t.Fatalf("CreateReceipt r-%d failed: %v", i, err)
		}
	}

	params := model.PaginationParams{Offset: 0, Limit: 2, SortBy: "upload_time", SortOrder: "DESC"}
	page, err := env.receipts.ListReceiptsByUser(env.ctx, "user-1", params)
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if page.TotalPages != 2 {
		t.Errorf("expected total_pages 2 (4 records / limit 2), got %d", page.TotalPages)
	}
}

func TestReceipt_Pagination_SortByPurchaseDate(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	for i, date := range []string{"2025.01.01", "2025.03.01", "2025.02.01"} {
		rec := env.sampleReceipt(fmt.Sprintf("r-%d", i+1), "user-1")
		rec.PurchaseDate = date
		if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
			t.Fatalf("CreateReceipt failed: %v", err)
		}
	}

	params := model.PaginationParams{Offset: 0, Limit: 10, SortBy: "purchase_date", SortOrder: "ASC"}
	page, err := env.receipts.ListReceiptsByUser(env.ctx, "user-1", params)
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if len(page.Data) != 3 {
		t.Fatalf("expected 3 receipts, got %d", len(page.Data))
	}
	if page.Data[0].PurchaseDate != "2025.01.01" {
		t.Errorf("expected first receipt date 2025.01.01, got %s", page.Data[0].PurchaseDate)
	}
	if page.Data[2].PurchaseDate != "2025.03.01" {
		t.Errorf("expected last receipt date 2025.03.01, got %s", page.Data[2].PurchaseDate)
	}
}

func TestReceipt_Delete(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	if err := env.receipts.DeleteReceipt(env.ctx, "r-1"); err != nil {
		t.Fatalf("DeleteReceipt failed: %v", err)
	}

	got, err := env.receipts.GetReceiptByID(env.ctx, "r-1")
	if err != nil {
		t.Fatalf("GetReceiptByID failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after delete, got receipt")
	}
}

func TestReceipt_DeleteNonexistent(t *testing.T) {
	env := newReceiptEnv(t)

	if err := env.receipts.DeleteReceipt(env.ctx, "nonexistent"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestReceipt_CascadeDeleteUser(t *testing.T) {
	env := newReceiptEnv(t)
	env.seedUser(t, "user-1")

	rec := env.sampleReceipt("r-1", "user-1")
	if err := env.receipts.CreateReceipt(env.ctx, rec); err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	// Delete the user — receipts should be cascade deleted.
	if err := env.users.DeleteUser(env.ctx, "user-1"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	got, err := env.receipts.GetReceiptByID(env.ctx, "r-1")
	if err != nil {
		t.Fatalf("GetReceiptByID failed: %v", err)
	}
	if got != nil {
		t.Fatal("expected receipt to be cascade deleted with user")
	}
}
