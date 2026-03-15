package model

// PaginationParams carries validated pagination input from the handler layer
// to the repository layer. All fields are set by the handler after validation;
// the repository must not re-validate them.
type PaginationParams struct {
	Offset    int    // >= 0
	Limit     int    // 1–100 (silently capped by handler)
	SortBy    string // DB column name (mapped from API param by handler allowlist)
	SortOrder string // "ASC" or "DESC" (normalised to uppercase by handler)
}

// Page is a generic paginated response envelope returned by all list endpoints.
type Page[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}
