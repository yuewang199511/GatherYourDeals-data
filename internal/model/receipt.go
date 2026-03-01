package model

import "errors"

// ErrFieldNotRegistered is returned when a receipt contains extra keys
// that are not defined in the meta table.
var ErrFieldNotRegistered = errors.New("one or more fields are not registered in the meta table")

// ErrMetaFieldExists is returned when trying to create a meta field that
// already exists.
var ErrMetaFieldExists = errors.New("field already exists")

// ErrFieldNotFound is returned when updating a field that does not exist.
var ErrFieldNotFound = errors.New("field not found")


// Receipt represents a single purchase record.
// Native fields are stored as dedicated columns; any user-defined fields
// are kept in the Extras JSON map.
type Receipt struct {
	ID           string             `json:"id"`
	ProductName  string             `json:"productName"`
	PurchaseDate string             `json:"purchaseDate"`
	Price        string             `json:"price"`
	Amount       string             `json:"amount"`
	StoreName    string             `json:"storeName"`
	Latitude     *float64           `json:"latitude,omitempty"`
	Longitude    *float64           `json:"longitude,omitempty"`
	Extras       map[string]interface{} `json:"extras,omitempty"`
	UploadTime   int64              `json:"uploadTime"`
	UserID       string             `json:"userId"`
}
