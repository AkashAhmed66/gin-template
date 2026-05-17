package dto

import "math"

// PageRequest captures pagination + sort query params parsed by web.BindPage.
type PageRequest struct {
	Page int    `form:"page"`
	Size int    `form:"size"`
	Sort string `form:"sort"` // "field,asc" | "field,desc" (one or comma-separated multiple)
}

// Normalize clamps Page/Size to sensible bounds. Use after binding.
func (p *PageRequest) Normalize() {
	if p.Page < 0 {
		p.Page = 0
	}
	if p.Size <= 0 {
		p.Size = 20
	}
	if p.Size > 200 {
		p.Size = 200
	}
}

// Offset returns the SQL offset corresponding to Page * Size.
func (p PageRequest) Offset() int { return p.Page * p.Size }

// Limit returns the SQL limit (alias of Size).
func (p PageRequest) Limit() int { return p.Size }

// PageResponse[T] mirrors Spring's Page<T> JSON shape so frontends written
// against the spring-boot template need no changes.
type PageResponse[T any] struct {
	Content       []T   `json:"content"`
	Page          int   `json:"page"`
	Size          int   `json:"size"`
	TotalElements int64 `json:"totalElements"`
	TotalPages    int   `json:"totalPages"`
	First         bool  `json:"first"`
	Last          bool  `json:"last"`
	Empty         bool  `json:"empty"`
}

// NewPage builds a PageResponse from a slice + the supplied PageRequest and
// total element count.
func NewPage[T any](content []T, req PageRequest, total int64) PageResponse[T] {
	if content == nil {
		content = []T{}
	}
	totalPages := 0
	if req.Size > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(req.Size)))
	}
	return PageResponse[T]{
		Content:       content,
		Page:          req.Page,
		Size:          req.Size,
		TotalElements: total,
		TotalPages:    totalPages,
		First:         req.Page == 0,
		Last:          req.Page >= totalPages-1,
		Empty:         len(content) == 0,
	}
}

// MapPage transforms a PageResponse[S] into a PageResponse[T] via a converter.
func MapPage[S any, T any](src PageResponse[S], fn func(S) T) PageResponse[T] {
	out := make([]T, len(src.Content))
	for i, v := range src.Content {
		out[i] = fn(v)
	}
	return PageResponse[T]{
		Content:       out,
		Page:          src.Page,
		Size:          src.Size,
		TotalElements: src.TotalElements,
		TotalPages:    src.TotalPages,
		First:         src.First,
		Last:          src.Last,
		Empty:         src.Empty,
	}
}
