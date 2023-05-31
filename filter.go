package nostr

// A Filter is a filter for subscription.
type Filter struct {
	IDs     []string    `json:"ids,omitempty"`
	Kinds   []EventKind `json:"kinds,omitempty"`
	Authors []string    `json:"authors,omitempty"`
	Tags    []Tag       `json:"-,omitempty"`
	Since   int64       `json:"since,omitempty"`
	Until   int64       `json:"until,omitempty"`
	Limit   int         `json:"limit,omitempty"`
	Search  string      `json:"search,omitempty"`
}
