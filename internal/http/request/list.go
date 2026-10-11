package request

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/helper/query"
)

// ParseList validates transport syntax only. Repositories own column whitelists;
// usecases own domain filter values. Unknown and duplicate parameters are errors.
func ParseList(r *http.Request, allowedFilters ...string) (query.Options, error) {
	if len(r.URL.RawQuery) > 8192 {
		return query.Options{}, query.Invalid("query string is too long")
	}
	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return query.Options{}, query.Invalid("invalid query string")
	}
	allowed := map[string]bool{"page": true, "page_size": true, "search": true, "sort": true, "order": true}
	for _, key := range allowedFilters {
		allowed[key] = true
	}
	for key, values := range params {
		if !allowed[key] {
			return query.Options{}, query.Invalid("unsupported query parameter")
		}
		if len(values) != 1 {
			return query.Options{}, query.Invalid("query parameters must not be repeated")
		}
	}
	o := query.Options{Page: query.DefaultPage, PageSize: query.DefaultPageSize, Search: params.Get("search"), Sort: params.Get("sort"), Order: query.Direction(params.Get("order")), Filters: map[string][]string{}}
	for _, key := range []string{"page", "page_size"} {
		if _, ok := params[key]; !ok {
			continue
		}
		value, err := strconv.Atoi(params.Get(key))
		if err != nil || value <= 0 {
			return query.Options{}, query.Invalid(key + " must be a positive integer")
		}
		if key == "page" {
			o.Page = value
		} else {
			o.PageSize = value
		}
	}
	for _, key := range []string{"sort", "order"} {
		if values, ok := params[key]; ok && strings.TrimSpace(values[0]) == "" {
			return query.Options{}, query.Invalid(key + " must not be empty")
		}
	}
	for _, key := range allowedFilters {
		value := strings.TrimSpace(params.Get(key))
		if value == "" {
			continue
		}
		values := strings.Split(value, ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
			if values[i] == "" {
				return query.Options{}, query.Invalid("filter values must not be empty")
			}
		}
		o.Filters[key] = values
	}
	return o.Normalize()
}
