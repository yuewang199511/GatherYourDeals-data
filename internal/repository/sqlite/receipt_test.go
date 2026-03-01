package sqlite_test

import (
	"context"
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

	list, err := env.receipts.ListReceiptsByUser(env.ctx, "user-1")
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 receipts for user-1, got %d", len(list))
	}

	list2, err := env.receipts.ListReceiptsByUser(env.ctx, "user-2")
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("expected 1 receipt for user-2, got %d", len(list2))
	}
}

func TestReceipt_ListByUser_Empty(t *testing.T) {
	env := newReceiptEnv(t)

	list, err := env.receipts.ListReceiptsByUser(env.ctx, "nobody")
	if err != nil {
		t.Fatalf("ListReceiptsByUser failed: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil for empty list, got %d items", len(list))
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
