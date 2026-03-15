package sqlite_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

// defaultMetaParams returns pagination params suitable for tests that don't need
// to test pagination specifically (fetches all, default sort).
func defaultMetaParams() model.PaginationParams {
	return model.PaginationParams{Offset: 0, Limit: 100, SortBy: "field_name", SortOrder: "ASC"}
}

func TestMetaField_NativeFieldsSeeded(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	page, err := repo.ListFields(ctx, defaultMetaParams())
	if err != nil {
		t.Fatalf("ListFields failed: %v", err)
	}

	expected := map[string]bool{
		"productName":  true,
		"purchaseDate": true,
		"price":        true,
		"amount":       true,
		"storeName":    true,
		"latitude":     true,
		"longitude":    true,
	}

	if page.Total != len(expected) {
		t.Fatalf("expected %d native fields, got total %d", len(expected), page.Total)
	}

	for _, f := range page.Data {
		if !expected[f.FieldName] {
			t.Errorf("unexpected field: %s", f.FieldName)
		}
		if !f.Native {
			t.Errorf("expected field %s to be native", f.FieldName)
		}
	}
}

func TestMetaField_CreateAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	field := &model.MetaField{
		FieldName:   "brand",
		Description: "brand of the product",
		FieldType:   "string",
		Native:      false,
	}

	if err := repo.CreateField(ctx, field); err != nil {
		t.Fatalf("CreateField failed: %v", err)
	}

	got, err := repo.GetField(ctx, "brand")
	if err != nil {
		t.Fatalf("GetField failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected field, got nil")
	}
	if got.FieldName != "brand" {
		t.Errorf("expected fieldName 'brand', got %q", got.FieldName)
	}
	if got.Description != "brand of the product" {
		t.Errorf("expected description 'brand of the product', got %q", got.Description)
	}
	if got.Native {
		t.Error("expected field to not be native")
	}
}

func TestMetaField_CreateDuplicate(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	field := &model.MetaField{
		FieldName:   "brand",
		Description: "brand of the product",
		FieldType:   "string",
	}

	if err := repo.CreateField(ctx, field); err != nil {
		t.Fatalf("first CreateField failed: %v", err)
	}

	err := repo.CreateField(ctx, field)
	if err == nil {
		t.Fatal("expected error on duplicate field, got nil")
	}
}

func TestMetaField_CreateDuplicate_NativeField(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	// Attempt to create a field with the same name as a native field.
	field := &model.MetaField{
		FieldName:   "productName",
		Description: "trying to overwrite native",
		FieldType:   "string",
	}

	err := repo.CreateField(ctx, field)
	if err == nil {
		t.Fatal("expected error when creating field with native name, got nil")
	}
}

func TestMetaField_GetNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	got, err := repo.GetField(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestMetaField_UpdateDescription(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	field := &model.MetaField{
		FieldName:   "brand",
		Description: "original",
		FieldType:   "string",
	}
	if err := repo.CreateField(ctx, field); err != nil {
		t.Fatalf("CreateField failed: %v", err)
	}

	if err := repo.UpdateDescription(ctx, "brand", "updated description"); err != nil {
		t.Fatalf("UpdateDescription failed: %v", err)
	}

	got, err := repo.GetField(ctx, "brand")
	if err != nil {
		t.Fatalf("GetField failed: %v", err)
	}
	if got.Description != "updated description" {
		t.Errorf("expected 'updated description', got %q", got.Description)
	}
}

func TestMetaField_UpdateDescription_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	err := repo.UpdateDescription(ctx, "nonexistent", "new desc")
	if err == nil {
		t.Fatal("expected error for nonexistent field, got nil")
	}
}

func TestMetaField_ListIncludesUserDefined(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	field := &model.MetaField{
		FieldName:   "brand",
		Description: "brand of the product",
		FieldType:   "string",
	}
	if err := repo.CreateField(ctx, field); err != nil {
		t.Fatalf("CreateField failed: %v", err)
	}

	page, err := repo.ListFields(ctx, defaultMetaParams())
	if err != nil {
		t.Fatalf("ListFields failed: %v", err)
	}

	// 7 native + 1 user-defined
	if page.Total != 8 {
		t.Fatalf("expected total 8, got %d", page.Total)
	}

	found := false
	for _, f := range page.Data {
		if f.FieldName == "brand" {
			found = true
			if f.Native {
				t.Error("expected 'brand' to not be native")
			}
		}
	}
	if !found {
		t.Error("expected to find 'brand' in list")
	}
}

func TestMetaField_Pagination_LimitOffset(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	// 7 native fields already seeded; add 3 more to get 10 total.
	for _, name := range []string{"brand", "color", "size"} {
		if err := repo.CreateField(ctx, &model.MetaField{
			FieldName: name, Description: name, FieldType: "string",
		}); err != nil {
			t.Fatalf("CreateField %s failed: %v", name, err)
		}
	}

	params := model.PaginationParams{Offset: 0, Limit: 3, SortBy: "field_name", SortOrder: "ASC"}
	page, err := repo.ListFields(ctx, params)
	if err != nil {
		t.Fatalf("ListFields failed: %v", err)
	}
	if page.Total != 10 {
		t.Errorf("expected total 10, got %d", page.Total)
	}
	if len(page.Data) != 3 {
		t.Errorf("expected 3 items in page, got %d", len(page.Data))
	}
	if page.TotalPages != 4 {
		t.Errorf("expected total_pages 4 (10/3 = ceil 4), got %d", page.TotalPages)
	}
}

func TestMetaField_Pagination_OffsetBeyondTotal(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	params := model.PaginationParams{Offset: 100, Limit: 10, SortBy: "field_name", SortOrder: "ASC"}
	page, err := repo.ListFields(ctx, params)
	if err != nil {
		t.Fatalf("ListFields failed: %v", err)
	}
	// 7 native fields are seeded.
	if page.Total != 7 {
		t.Errorf("expected total 7, got %d", page.Total)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected empty data when offset > total, got %d items", len(page.Data))
	}
}

func TestMetaField_Pagination_SortDesc(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	params := model.PaginationParams{Offset: 0, Limit: 2, SortBy: "field_name", SortOrder: "DESC"}
	page, err := repo.ListFields(ctx, params)
	if err != nil {
		t.Fatalf("ListFields failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(page.Data))
	}
	// In DESC order, the last alphabetically should be first.
	if page.Data[0].FieldName <= page.Data[1].FieldName {
		t.Errorf("expected descending order, got %q then %q", page.Data[0].FieldName, page.Data[1].FieldName)
	}
}
