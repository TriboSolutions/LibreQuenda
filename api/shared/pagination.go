package shared

import (
	"net/http"
	"strconv"

	"github.com/angelbarreiros/Penguin/router/helpers"
)

// DefaultPageSize: a list's page when the client asks for none, and the most it may ask for.
const DefaultPageSize uint = 100

// MaxPage: the last page a list may ask for. Far beyond any list here, and it keeps
// the offset (page × size) within the database's INT, which would otherwise overflow.
const MaxPage = 10000

// Page: ?page= (from 1) and ?pageSize= (1..100, 100 by default), as Penguin reads
// them. Penguin ignores bad values; here they are an error, so a client hears
// about its mistake instead of getting another page.
func Page(r *http.Request) (helpers.PaginationParams, *ErrorResponse) {
	return PageOf(r, DefaultPageSize)
}

// ListPageSize: the councils' lists and histories (events, announcements, polls,
// the maintenance list) a page at a time, so none loads all of it.
const ListPageSize uint = 20

// PageOf: Page, with size as the default ?pageSize= (still 100 at most).
func PageOf(r *http.Request, size uint) (helpers.PaginationParams, *ErrorResponse) {
	var q = r.URL.Query()
	for _, key := range []string{"page", "pageSize"} {
		if text := q.Get(key); text != "" {
			if value, err := strconv.ParseUint(text, 10, 31); err != nil || value < 1 || (key == "page" && value > MaxPage) || (key == "pageSize" && value > uint64(DefaultPageSize)) {
				return helpers.PaginationParams{}, &ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
			}
		}
	}
	return helpers.GetPaginationParamsWithMax(r, size, DefaultPageSize), nil
}

// PageDto: where a page sits in the whole list.
type PageDto struct {
	Page     uint  `json:"page"`
	PageSize uint  `json:"page_size"`
	Total    int64 `json:"total"`
	HasMore  bool  `json:"has_more"`
}

// NewPageDto for a page of params out of total items.
func NewPageDto(params helpers.PaginationParams, total int64) PageDto {
	return PageDto{Page: params.Page, PageSize: params.PageSize, Total: total, HasMore: int64(params.Offset()+params.PageSize) < total}
}

// Slice: the page of a list held in memory (already filtered and sorted).
func Slice[T any](items []T, params helpers.PaginationParams) []T {
	var start = int(params.Offset())
	if start >= len(items) {
		return []T{}
	}
	return items[start:min(start+int(params.PageSize), len(items))]
}

// Rows: the LIMIT and OFFSET of a page, for the queries.
func Rows(params helpers.PaginationParams) (int32, int32) {
	return int32(params.PageSize), int32(params.Offset())
}
