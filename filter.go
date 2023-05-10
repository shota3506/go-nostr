package nostr

type Filter struct {
	IDs     []string    `json:"ids,omitempty"`
	Kinds   []EventKind `json:"kinds,omitempty"`
	Authors []string    `json:"authors,omitempty"`
	Tags    []Tag       `json:"-,omitempty"`
	Since   int64       `json:"since,omitempty"` // TODO: use box type
	Until   int64       `json:"until,omitempty"` // TODO: use box type
	Limit   int         `json:"limit,omitempty"`
	Search  string      `json:"search,omitempty"`
}
