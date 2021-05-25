package filters

import (
	"fmt"
	"math"
	"strings"
)

type Input struct {
	Page                 int
	PageSize             int
	SortCol              string
	SortSafeList         []string
	Search               string
	SearchCol            string
	SearchColumnSafeList []string
}

// Check that the client-provided SortCol field matches one of the entries in our sortList
// and if it does, extract the column name from the SortCol field by stripping the leading
// hyphen character (if one exists).
func (p Input) SortColumn() string {
	return strings.TrimPrefix(p.SortCol, "-")
}

// Return the sort direction ("ASC" or "DESC") depending on the prefix character of the
// SortCol field.
func (p Input) SortDirection() string {
	if strings.HasPrefix(p.SortCol, "-") {
		return "DESC"
	}
	return "ASC"
}

func (p Input) Limit() int {
	return p.PageSize
}

func (p Input) Offset() int {
	return (p.Page - 1) * p.PageSize
}

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

type Meta struct {
	CurrentPage  int    `json:"current_page"`
	PageSize     int    `json:"page_size"`
	FirstPage    int    `json:"first_page"`
	LastPage     int    `json:"last_page"`
	TotalRecords int64  `json:"total_records"`
	Search       string `json:"search,omitempty"`
	SearchField  string `json:"search_field,omitempty"`
}

// The CalculateOutput() function calculates the appropriate pagination metadata
// values given the total number of records, current page, and page size values. Note
// that the last page value is calculated using the math.Ceil() function, which rounds
// up a float to the nearest integer. So, for example, if there were 12 records in total
// and a page size of 5, the last page value would be math.Ceil(12/5) = 3.
func (p Input) CalculateOutput(totalRecords int64) Meta {
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
