package domain

// BaseFetchOptions contains common pagination/sorting fields used by fetch options.
type BaseFetchOptions struct {
	// Page number (1-based). If 0, pagination is disabled and Limit/Offset are used directly.
	Page int
	// Limit number of items per page. If 0, a sensible default should be applied by callers.
	Limit int
	// Offset for results. When Page is set, Offset is calculated as (Page-1)*Limit.
	Offset int
	// Sort expression, e.g. "created_at desc" or "username asc".
	Sort string
	// Query is a free-text search/filter string.
	Query string
}
