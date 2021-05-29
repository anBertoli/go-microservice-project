package filters

import (
	"fmt"
	"math"
	"strings"
)

// The filters package provides utilities to be used in listing operations,
// to support easily pagination and filtering.

// Filtering and pagination input for listing operations.
type Input struct {
	Page                 int
	PageSize             int
	SortCol              string
	SortSafeList         []string
	Search               string
	SearchCol            string
	SearchColumnSafeList []string
}

// Extract the column to be used for sorting.
func (p Input) SortColumn() string {
	return strings.TrimPrefix(p.SortCol, "-")
}

// Get sort direction ("ASC" or "DESC") to be used during SELECT queries
// depending on the prefix character of the SortCol field.
func (p Input) SortDirection() string {
	if strings.HasPrefix(p.SortCol, "-") {
		return "DESC"
	}
	return "ASC"
}

// Limit to be used during database SELECT queries.
func (p Input) Limit() int {
	return p.PageSize
}

// Offset to be used during database SELECT queries.
func (p Input) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Make sure the filter input is valid, that is, the sortCol is valid (must be present in
// SortSafeList) and the searchCol is valid (contained in the SearchColumnSafeList).
func (p Input) Validate() error {
	var ok bool
	for _, safeValue := range p.SortSafeList {
		if p.SortCol == safeValue {
			ok = true
		}
	}
	if !ok {
		return fmt.Errorf("%s not allowed as ordering parameter", p.SortCol)
	}
	for _, searchCol := range p.SearchColumnSafeList {
		if p.SearchCol == searchCol {
			return nil
		}
	}
	return fmt.Errorf("%s not allowed as search column", p.SearchCol)
}

// Metadata output of a listing operation, based upon the Input and
// the result of the listing operation.
type Meta struct {
	CurrentPage  int    `json:"current_page"`
	PageSize     int    `json:"page_size"`
	FirstPage    int    `json:"first_page"`
	LastPage     int    `json:"last_page"`
	TotalRecords int64  `json:"total_records"`
	Search       string `json:"search,omitempty"`
	SearchField  string `json:"search_field,omitempty"`
}

// The CalculateMetadata() function calculates the appropriate pagination metadata given
// the number of obtained records, current page, and page size values.
func (p Input) CalculateMetadata(totalRecords int64) Meta {
	searchField := ""
	if p.Search != "" {
		searchField = p.SearchCol
	}
	meta := Meta{
		Search:       p.Search,
		SearchField:  searchField,
		PageSize:     p.PageSize,
		CurrentPage:  p.Page,
		FirstPage:    1,
		LastPage:     1,
		TotalRecords: totalRecords,
	}
	if totalRecords != 0 {
		meta.LastPage = int(math.Ceil(float64(totalRecords) / float64(p.PageSize)))
	}
	return meta
}
