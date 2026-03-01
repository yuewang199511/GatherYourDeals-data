package sqlite_test

import (
	"context"
	"testing"

	"github.com/gatheryourdeals/data/internal/model"
	"github.com/gatheryourdeals/data/internal/repository/sqlite"
	"github.com/gatheryourdeals/data/internal/repository/sqlite/testutil"
)

func TestMetaField_NativeFieldsSeeded(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := sqlite.NewMetaFieldRepo(db)
	ctx := context.Background()

	fields, err := repo.ListFields(ctx)
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

	if len(fields) != len(expected) {
		t.Fatalf("expected %d native fields, got %d", len(expected), len(fields))
	}

	for _, f := range fields {
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

	fields, err := repo.ListFields(ctx)
	if err != nil {
		t.Fatalf("ListFields failed: %v", err)
	}

	// 7 native + 1 user-defined
	if len(fields) != 8 {
		t.Fatalf("expected 8 fields, got %d", len(fields))
	}

	found := false
	for _, f := range fields {
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
