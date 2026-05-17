package product

import (
	"strconv"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/AkashAhmed66/gin-template/internal/modules/audit"
	"github.com/AkashAhmed66/gin-template/internal/modules/file"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc   Service
	files file.Service
}

func NewHandler(svc Service, files file.Service) *Handler {
	return &Handler{svc: svc, files: files}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/products")
	g.GET("", security.HasPermission(permission.ProductRead), web.Handler(h.list))
	g.GET("/:id", security.HasPermission(permission.ProductRead), web.Handler(h.get))
	g.POST("", security.HasPermission(permission.ProductWrite), web.Handler(h.create))
	g.PUT("/:id", security.HasPermission(permission.ProductWrite), web.Handler(h.update))
	g.POST("/:id/image", security.HasPermission(permission.ProductWrite), web.Handler(h.uploadImage))
	g.DELETE("/:id", security.HasPermission(permission.ProductDelete), web.Handler(h.delete))
}

// list godoc
// @Summary      List products
// @Description  Paginated search. Supports q, status, minPrice, maxPrice + page/size/sort.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        q         query     string  false  "free-text query"
// @Param        status    query     string  false  "DRAFT|PUBLISHED|ARCHIVED"
// @Param        minPrice  query     number  false  "min price"
// @Param        maxPrice  query     number  false  "max price"
// @Param        page      query     int     false  "page index (0-based)"
// @Param        size      query     int     false  "page size (max 200)"
// @Success      200       {object}  dto.ApiResponse{data=dto.PageResponse[product.Response]}
// @Router       /api/v1/products [get]
func (h *Handler) list(c *gin.Context) (*dto.ApiResponse, error) {
	var f Filter
	_ = c.ShouldBindQuery(&f)
	page := web.BindPage(c)
	res, err := h.svc.Search(c.Request.Context(), f, page)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// get godoc
// @Summary      Get product by id
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "product id"
// @Success      200  {object}  dto.ApiResponse{data=product.Response}
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/products/{id} [get]
func (h *Handler) get(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	res, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// create godoc
// @Summary      Create product
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      Request  true  "product payload"
// @Success      201   {object}  dto.ApiResponse{data=product.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      409   {object}  dto.ApiResponse
// @Router       /api/v1/products [post]
func (h *Handler) create(c *gin.Context) (*dto.ApiResponse, error) {
	audit.Action(c, "PRODUCT_CREATE", "Product", "")
	var req Request
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		return nil, err
	}
	return dto.Created(res), nil
}

// update godoc
// @Summary      Update product
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int      true  "product id"
// @Param        body  body      Request  true  "product payload"
// @Success      200   {object}  dto.ApiResponse{data=product.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Failure      409   {object}  dto.ApiResponse
// @Router       /api/v1/products/{id} [put]
func (h *Handler) update(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	audit.Action(c, "PRODUCT_UPDATE", "Product", strconv.FormatUint(id, 10))
	var req Request
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// uploadImage godoc
// @Summary      Upload product image
// @Description  Multipart form with field "file". URL is stored on the product.
// @Tags         products
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int   true  "product id"
// @Param        file  formData  file  true  "image file"
// @Success      200   {object}  dto.ApiResponse{data=product.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Router       /api/v1/products/{id}/image [post]
func (h *Handler) uploadImage(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	audit.Action(c, "PRODUCT_IMAGE_UPLOAD", "Product", strconv.FormatUint(id, 10))
	fh, err := c.FormFile("file")
	if err != nil {
		return nil, errs.BadRequest("Field 'file' is required")
	}
	uploaded, err := h.files.Save(fh, "products")
	if err != nil {
		return nil, err
	}
	// Apply via Update to ensure audit + UpdatedAt fire.
	existing, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	req := Request{
		Name: existing.Name, SKU: existing.SKU, Description: existing.Description,
		Price: existing.Price, Stock: existing.Stock,
		ImageURL: uploaded.URL, Status: existing.Status,
	}
	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// delete godoc
// @Summary      Delete product
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "product id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/products/{id} [delete]
func (h *Handler) delete(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	audit.Action(c, "PRODUCT_DELETE", "Product", strconv.FormatUint(id, 10))
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		return nil, err
	}
	return dto.Message("Deleted"), nil
}

func parseID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, errs.BadRequest("Invalid id")
	}
	return id, nil
}
