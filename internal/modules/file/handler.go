package file

import (
	"net/http"
	"path"
	"strings"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// Register wires file routes. uploadAuth must be the require-auth middleware.
//
//   - POST /files/upload   — authenticated, multipart with field "file".
//   - GET  /files/*path    — public; supports /files/x.jpg and /files/sub/x.jpg.
//
// We use a single catch-all so Gin's router accepts both shapes (it rejects
// having both `/files/:subfolder/:name` and `/files/:name` at the same level).
func (h *Handler) Register(r *gin.RouterGroup, uploadAuth gin.HandlerFunc) {
	g := r.Group("/files")
	g.POST("/upload", uploadAuth, web.Handler(h.upload))
	g.GET("/*path", h.serve)
}

// upload godoc
// @Summary      Upload a file
// @Description  Multipart upload with field "file" and optional "subfolder". Stored filenames become <uuid>.<ext>.
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file       formData  file    true   "file to upload"
// @Param        subfolder  formData  string  false  "optional subfolder under the storage root"
// @Success      201        {object}  dto.ApiResponse{data=file.Response}
// @Failure      400        {object}  dto.ApiResponse
// @Failure      401        {object}  dto.ApiResponse
// @Router       /api/v1/files/upload [post]
func (h *Handler) upload(c *gin.Context) (*dto.ApiResponse, error) {
	fh, err := c.FormFile("file")
	if err != nil {
		return nil, errs.BadRequest("Field 'file' is required")
	}
	sub := c.PostForm("subfolder")
	res, err := h.svc.Save(fh, sub)
	if err != nil {
		return nil, err
	}
	return dto.Created(res), nil
}

// serve godoc
// @Summary      Serve a stored file
// @Description  Public route — used by <img> tags. Supports /files/x.jpg and /files/sub/dir/x.jpg.
// @Tags         files
// @Produce      octet-stream
// @Param        path  path      string  true  "subfolder/filename (or just filename)"
// @Success      200   {file}    binary
// @Failure      404   {object}  dto.ApiResponse
// @Router       /api/v1/files/{path} [get]
//
// serve splits the catch-all into (subfolder, name). The catch-all value
// always starts with "/" — strip it, then split on the last separator.
func (h *Handler) serve(c *gin.Context) {
	raw := strings.TrimPrefix(c.Param("path"), "/")
	if raw == "" {
		web.WriteResponse(c, dto.Error(http.StatusNotFound, "File not found", nil))
		return
	}
	dir, name := path.Split(raw)
	sub := strings.TrimSuffix(dir, "/")

	resolved, err := h.svc.ResolvePath(sub, name)
	if err != nil {
		web.WriteResponse(c, dto.Error(http.StatusNotFound, "File not found", nil))
		return
	}
	c.File(resolved)
}
