package model

// MetaField describes a data field — either a native (built-in) field or a
// user-defined custom field. Native fields cannot be deleted or renamed.
type MetaField struct {
	FieldName   string `json:"fieldName"`
	Description string `json:"description"`
	FieldType   string `json:"type"`
	Native      bool   `json:"native"`
}
