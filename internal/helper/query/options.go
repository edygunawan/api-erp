// Package query defines bounded list options and pagination, independently of
// HTTP, database drivers, authentication, and SQL construction.
package query

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxSearchLength = 256
	MaxFilterValues = 100
	MaxFilterLength = 256
)

type Direction string

const (
	ASC  Direction = "ASC"
	DESC Direction = "DESC"
)

type Options struct {
	Page     int
	PageSize int
	Search   string
	Sort     string
	Order    Direction
	// A single value is exact matching; multiple values use IN. No client operators.
	Filters map[string][]string
}

// ValidationError describes invalid client options, never configuration or SQL.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }
func Invalid(message string) error       { return &ValidationError{Message: message} }

// Normalize gives zero-value internal callers defaults. HTTP parsing separately
// rejects explicitly supplied zero page/page_size values rather than hiding them.
func (o Options) Normalize() (Options, error) {
	if o.Page == 0 {
		o.Page = DefaultPage
	}
	if o.PageSize == 0 {
		o.PageSize = DefaultPageSize
	}
	if o.Page < 1 {
		return Options{}, Invalid("page must be a positive integer")
	}
	if o.PageSize < 1 || o.PageSize > MaxPageSize {
		return Options{}, Invalid(fmt.Sprintf("page_size must be between 1 and %d", MaxPageSize))
	}
	if int64(o.Page-1) > math.MaxInt64/int64(o.PageSize) {
		return Options{}, Invalid("pagination offset is too large")
	}
	o.Search = strings.TrimSpace(o.Search)
	if !utf8.ValidString(o.Search) || utf8.RuneCountInString(o.Search) > MaxSearchLength {
		return Options{}, Invalid("search must contain at most 256 characters")
	}
	o.Sort = strings.TrimSpace(o.Sort)
	o.Order = Direction(strings.ToUpper(strings.TrimSpace(string(o.Order))))
	if o.Order != "" && o.Order != ASC && o.Order != DESC {
		return Options{}, Invalid("order must be asc or desc")
	}
	filters := make(map[string][]string, len(o.Filters))
	n := 0
	for key, values := range o.Filters {
		filters[key] = nil
		if key == "" {
			return Options{}, Invalid("invalid filter")
		}
		n += len(values)
		if n > MaxFilterValues {
			return Options{}, Invalid("too many filter values")
		}
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if !utf8.ValidString(value) || utf8.RuneCountInString(value) > MaxFilterLength {
				return Options{}, Invalid("filter value must contain at most 256 characters")
			}
			filters[key] = append(filters[key], value)
		}
	}
	o.Filters = filters
	return o, nil
}

func (o Options) Offset() int64 { return int64(o.Page-1) * int64(o.PageSize) }

type Meta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type Page[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}

// NewPage consumes normalized options and a non-negative COUNT(*) result.
// Requested out-of-range pages are preserved. Empty data always encodes as [].
func NewPage[T any](data []T, o Options, total int64) Page[T] {
	if data == nil {
		data = []T{}
	}
	pages := total / int64(o.PageSize)
	if total%int64(o.PageSize) != 0 {
		pages++
	}
	return Page[T]{Data: data, Meta: Meta{Page: o.Page, PageSize: o.PageSize, Total: total, TotalPages: pages}}
}
