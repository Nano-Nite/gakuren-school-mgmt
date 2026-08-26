package model

type CountResult struct {
	Count int `db:"count"`
}

type DataStatistics struct {
	CurrentRow  int `json:"current_row"`
	StartRow    int `json:"start_row"`
	EndRow      int `json:"end_row"`
	TotalRow    int `json:"total_row"`
	CurrentPage int `json:"current_page"`
	MaxPage     int `json:"max_page"`
	RowPerPage  int `json:"row_per_page"`
}

type SearchPayload struct {
	Search     *string                   `json:"search,omitempty"`
	Filter     *ReadClassModelResult     `json:"filter,omitempty"`
	Page       *int                      `json:"page,omitempty"`
	RowPerPage *int                      `json:"row_per_page,omitempty"`
	SortBy     *[]map[string]interface{} `json:"sort_by,omitempty"`
}
