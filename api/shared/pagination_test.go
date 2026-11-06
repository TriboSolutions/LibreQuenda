package shared

import (
	"net/http/httptest"
	"testing"
)

func TestPage(t *testing.T) {
	var cases = []struct {
		query          string
		page, pageSize uint
		ok             bool
	}{
		{"", 1, 100, true},
		{"page=3&pageSize=20", 3, 20, true},
		{"page=0", 0, 0, false},
		{"page=x", 0, 0, false},
		{"pageSize=101", 0, 0, false},
		{"page=10000", 10000, 100, true},
		{"page=10001", 0, 0, false}, // the offset would overflow the database's INT
		{"pageSize=-5", 0, 0, false},
	}
	for _, c := range cases {
		var p, err = Page(httptest.NewRequest("GET", "/x?"+c.query, nil))
		if (err == nil) != c.ok || (c.ok && (p.Page != c.page || p.PageSize != c.pageSize)) {
			t.Errorf("%q: %+v %v", c.query, p, err)
		}
	}
}

func TestSliceAndPageDto(t *testing.T) {
	var items = []int{1, 2, 3, 4, 5}
	var p, _ = Page(httptest.NewRequest("GET", "/x?page=2&pageSize=2", nil))
	if got := Slice(items, p); len(got) != 2 || got[0] != 3 {
		t.Errorf("page 2 of 2 = %v", got)
	}
	if dto := NewPageDto(p, 5); !dto.HasMore || dto.Total != 5 {
		t.Errorf("dto = %+v", dto)
	}
	var last, _ = Page(httptest.NewRequest("GET", "/x?page=3&pageSize=2", nil))
	if dto := NewPageDto(last, 5); dto.HasMore {
		t.Error("the last page has no more")
	}
	var past, _ = Page(httptest.NewRequest("GET", "/x?page=9&pageSize=2", nil))
	if got := Slice(items, past); len(got) != 0 {
		t.Errorf("past the end = %v", got)
	}
}
