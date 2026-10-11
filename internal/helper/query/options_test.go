package query

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestOptionsPagination(t *testing.T) {
	for _, tc := range []struct {
		name       string
		input      Options
		page, size int
		offset     int64
		invalid    bool
	}{
		{"defaults", Options{}, 1, 20, 0, false},
		{"custom", Options{Page: 3, PageSize: 25}, 3, 25, 50, false},
		{"maximum", Options{Page: 2, PageSize: 100}, 2, 100, 100, false},
		{"negative page", Options{Page: -1}, 0, 0, 0, true},
		{"negative size", Options{PageSize: -1}, 0, 0, 0, true},
		{"oversized", Options{PageSize: 101}, 0, 0, 0, true},
		{"overflow", Options{Page: math.MaxInt, PageSize: 100}, 0, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.input.Normalize()
			if tc.invalid {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil || actual.Page != tc.page || actual.PageSize != tc.size || actual.Offset() != tc.offset {
				t.Fatalf("options=%+v err=%v", actual, err)
			}
		})
	}
}

func TestNormalizeBoundsAndOwnership(t *testing.T) {
	input := Options{Search: "  person  ", Order: "desc", Filters: map[string][]string{"status": {" ACTIVE ", ""}}}
	actual, err := input.Normalize()
	if err != nil || actual.Search != "person" || actual.Order != DESC || !reflect.DeepEqual(actual.Filters["status"], []string{"ACTIVE"}) {
		t.Fatalf("options=%+v err=%v", actual, err)
	}
	actual.Filters["status"][0] = "INACTIVE"
	if input.Filters["status"][0] != " ACTIVE " {
		t.Fatal("normalization mutated caller data")
	}
	for _, opts := range []Options{
		{Order: "desc;--"}, {Search: strings.Repeat("é", 257)}, {Search: string([]byte{0xff})},
		{Filters: map[string][]string{"status": make([]string, 101)}},
		{Filters: map[string][]string{"module": {strings.Repeat("a", 257)}}},
	} {
		if _, err := opts.Normalize(); err == nil {
			t.Fatalf("invalid options accepted: %+v", opts)
		}
	}
	if _, err := (Options{Search: strings.Repeat("é", 256)}).Normalize(); err != nil {
		t.Fatal("unicode limit counts bytes instead of characters")
	}
}

func TestPageMetadataAndEmptyJSON(t *testing.T) {
	o, _ := (Options{Page: 999, PageSize: 20}).Normalize()
	for _, tc := range []struct{ total, pages int64 }{{0, 0}, {1, 1}, {19, 1}, {20, 1}, {21, 2}, {101, 6}, {math.MaxInt64, 461168601842738791}} {
		result := NewPage[string](nil, o, tc.total)
		if result.Data == nil || result.Meta.Page != 999 || result.Meta.PageSize != 20 || result.Meta.Total != tc.total || result.Meta.TotalPages != tc.pages {
			t.Fatalf("page=%+v", result)
		}
		encoded, err := json.Marshal(result)
		if err != nil || !strings.Contains(string(encoded), `"data":[]`) {
			t.Fatalf("empty list json=%s err=%v", encoded, err)
		}
	}
}
