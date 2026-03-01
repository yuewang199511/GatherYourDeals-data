package model

import (
	"encoding/json"
	"errors"
)

// ErrFieldNotRegistered is returned when a receipt contains extra keys
// that are not defined in the meta table.
var ErrFieldNotRegistered = errors.New("one or more fields are not registered in the meta table")

// ErrMetaFieldExists is returned when trying to create a meta field that
// already exists.
var ErrMetaFieldExists = errors.New("field already exists")

// ErrFieldNotFound is returned when updating a field that does not exist.
var ErrFieldNotFound = errors.New("field not found")

// nativeFieldSet is the set of field names that are stored as dedicated columns.
var nativeFieldSet = map[string]bool{
	"productName":  true,
	"purchaseDate": true,
	"price":        true,
	"amount":       true,
	"storeName":    true,
	"latitude":     true,
	"longitude":    true,
}

// IsNativeField returns true if the field name is a native (built-in) column.
func IsNativeField(name string) bool {
	return nativeFieldSet[name]
}

// Receipt represents a single purchase record.
// Native fields are stored as dedicated columns; any user-defined fields
// are kept in the Extras map internally but serialized flat in JSON.
type Receipt struct {
	ID           string                 `json:"-"`
	ProductName  string                 `json:"-"`
	PurchaseDate string                 `json:"-"`
	Price        string                 `json:"-"`
	Amount       string                 `json:"-"`
	StoreName    string                 `json:"-"`
	Latitude     *float64               `json:"-"`
	Longitude    *float64               `json:"-"`
	Extras       map[string]interface{} `json:"-"`
	UploadTime   int64                  `json:"-"`
	UserID       string                 `json:"-"`
}

// MarshalJSON produces a flat JSON object merging native fields and extras.
func (r *Receipt) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"id":           r.ID,
		"productName":  r.ProductName,
		"purchaseDate": r.PurchaseDate,
		"price":        r.Price,
		"amount":       r.Amount,
		"storeName":    r.StoreName,
		"uploadTime":   r.UploadTime,
		"userId":       r.UserID,
	}
	if r.Latitude != nil {
		m["latitude"] = *r.Latitude
	}
	if r.Longitude != nil {
		m["longitude"] = *r.Longitude
	}
	for k, v := range r.Extras {
		m[k] = v
	}
	return json.Marshal(m)
}

// ParseReceiptFromMap builds a Receipt from a flat key-value map.
// Native fields are extracted into struct fields; everything else
// goes into Extras. Returns the receipt and a list of unknown keys
// (keys that are neither native nor in the extras map — the caller
// should validate these against the meta table).
func ParseReceiptFromMap(m map[string]interface{}) (*Receipt, map[string]interface{}) {
	r := &Receipt{
		Extras: make(map[string]interface{}),
	}

	if v, ok := m["productName"].(string); ok {
		r.ProductName = v
	}
	if v, ok := m["purchaseDate"].(string); ok {
		r.PurchaseDate = v
	}
	if v, ok := m["price"].(string); ok {
		r.Price = v
	}
	if v, ok := m["amount"].(string); ok {
		r.Amount = v
	}
	if v, ok := m["storeName"].(string); ok {
		r.StoreName = v
	}
	if v, ok := m["latitude"].(float64); ok {
		r.Latitude = &v
	}
	if v, ok := m["longitude"].(float64); ok {
		r.Longitude = &v
	}

	// Track server-managed fields so we skip them.
	// prevents injection
	skip := map[string]bool{
		"id": true, "uploadTime": true, "userId": true,
	}

	extras := make(map[string]interface{})
	for k, v := range m {
		if nativeFieldSet[k] || skip[k] {
			continue
		}
		extras[k] = v
	}

	return r, extras
}
