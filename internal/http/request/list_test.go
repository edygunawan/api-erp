package request

import (
	"errors"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

func TestParseListDefaultsAndValues(t *testing.T) {
	o, err := ParseList(httptest.NewRequest("GET", "/users", nil), "status")
	if err != nil || o.Page != 1 || o.PageSize != 20 {
		t.Fatalf("options=%+v err=%v", o, err)
	}
	o, err = ParseList(httptest.NewRequest("GET", "/users?page=2&page_size=100&search=+Jane+&sort=full_name&order=desc&status=ACTIVE,INACTIVE", nil), "status")
	if err != nil || o.Page != 2 || o.PageSize != 100 || o.Offset() != 100 || o.Search != "Jane" || o.Sort != "full_name" || o.Order != query.DESC || !reflect.DeepEqual(o.Filters["status"], []string{"ACTIVE", "INACTIVE"}) {
		t.Fatalf("options=%+v err=%v", o, err)
	}
	o, err = ParseList(httptest.NewRequest("GET", "/users?status=&search=", nil), "status")
	if err != nil || len(o.Filters) != 0 || o.Search != "" {
		t.Fatalf("empty values: %+v %v", o, err)
	}
}
func TestParseListRejectsMalformedAndUnboundedQueries(t *testing.T) {
	for _, raw := range []string{
		"page=0", "page=-1", "page=abc", "page=", "page_size=0", "page_size=101", "page_size=1000000",
		"page=999999999999999999999999", "page=9223372036854775807&page_size=100",
		"page=1&page=2", "status=ACTIVE&status=INACTIVE", "unexpected=abc", "status[operator]=ACTIVE",
		"sort=", "order=", "order=desc%3B--", "page=1;page_size=20", "search=%ZZ", "status=ACTIVE,,INACTIVE",
		"search=" + url.QueryEscape(strings.Repeat("é", 257)), "search=" + strings.Repeat("x", 8193),
		"status=" + strings.Repeat("ACTIVE,", 101) + "INACTIVE",
	} {
		t.Run(raw[:min(len(raw), 40)], func(t *testing.T) {
			r := httptest.NewRequest("GET", "/users", nil)
			r.URL.RawQuery = raw
			_, err := ParseList(r, "status")
			var invalid *query.ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("raw=%q err=%v", raw, err)
			}
		})
	}
}
