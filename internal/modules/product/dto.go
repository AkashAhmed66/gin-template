package product

import "github.com/AkashAhmed66/gin-template/internal/common/dto"

type Request struct {
	Name        string  `json:"name" binding:"required,min=2,max=200"`
	SKU         string  `json:"sku" binding:"required,min=1,max=100"`
	Description string  `json:"description" binding:"max=10000"`
	Price       float64 `json:"price" binding:"required,gte=0"`
	Stock       int     `json:"stock" binding:"gte=0"`
	ImageURL    string  `json:"imageUrl" binding:"max=500"`
	Status      Status  `json:"status" binding:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
}

type Response struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	ImageURL    string  `json:"imageUrl,omitempty"`
	Status      Status  `json:"status"`
	dto.BaseResponse
}

type Filter struct {
	Q      string `form:"q"`
	Status Status `form:"status"`
	MinP   *float64 `form:"minPrice"`
	MaxP   *float64 `form:"maxPrice"`
}

func ToResponse(p Product) Response {
	r := Response{
		ID:          p.ID,
		Name:        p.Name,
		SKU:         p.SKU,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		ImageURL:    p.ImageURL,
		Status:      p.Status,
	}
	r.CreatedAt = p.CreatedAt
	r.UpdatedAt = p.UpdatedAt
	r.CreatedBy = p.CreatedBy
	r.UpdatedBy = p.UpdatedBy
	if p.DeletedAt.Valid {
		t := p.DeletedAt.Time
		r.DeletedAt = &t
		r.DeletedBy = p.DeletedBy
	}
	return r
}

func ToResponses(ps []Product) []Response {
	out := make([]Response, len(ps))
	for i, p := range ps {
		out[i] = ToResponse(p)
	}
	return out
}
