// Package file is the disk-backed upload + serve module.
//
//   - POST /api/v1/files/upload — multipart upload (authenticated)
//   - GET  /api/v1/files/:sub/:name — serve a stored file (public, so <img>
//     tags work without a token).
//
// Filenames become `<uuid>.<ext>` so uploads can't overwrite each other; the
// subfolder is sanitized against `..` so path traversal is impossible.
package file

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/google/uuid"
)

type Response struct {
	URL          string `json:"url"`
	StoredName   string `json:"storedName"`
	OriginalName string `json:"originalName"`
	Subfolder    string `json:"subfolder,omitempty"`
	Size         int64  `json:"size"`
	ContentType  string `json:"contentType,omitempty"`
}

// Service is the upload/lookup interface.
type Service interface {
	Save(fh *multipart.FileHeader, subfolder string) (*Response, error)
	ResolvePath(subfolder, name string) (string, error)
}

type service struct {
	basePath      string
	publicBaseURL string
	maxSize       int64
}

// New returns a file service that stores uploads under cfg.BasePath.
func New(cfg config.StorageConfig, publicBaseURL string) Service {
	_ = os.MkdirAll(cfg.BasePath, 0o755)
	return &service{
		basePath:      cfg.BasePath,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/") + "/api/v1/files",
		maxSize:       cfg.MaxUploadSize,
	}
}

// Save writes fh under sanitized(subfolder)/<uuid>.<ext> and returns the
// public URL + metadata.
func (s *service) Save(fh *multipart.FileHeader, subfolder string) (*Response, error) {
	if fh == nil {
		return nil, errs.BadRequest("No file uploaded")
	}
	if s.maxSize > 0 && fh.Size > s.maxSize {
		return nil, errs.BadRequest(fmt.Sprintf("File exceeds %d bytes", s.maxSize))
	}
	sub := sanitize(subfolder)
	dir := filepath.Join(s.basePath, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	stored := uuid.NewString() + ext
	dst := filepath.Join(dir, stored)

	src, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return nil, err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, src); err != nil {
		_ = os.Remove(dst)
		return nil, err
	}

	url := fmt.Sprintf("%s/%s/%s", s.publicBaseURL, sub, stored)
	if sub == "" {
		url = fmt.Sprintf("%s/%s", s.publicBaseURL, stored)
	}
	contentType := ""
	if fh.Header != nil {
		contentType = fh.Header.Get("Content-Type")
	}
	return &Response{
		URL:          url,
		StoredName:   stored,
		OriginalName: fh.Filename,
		Subfolder:    sub,
		Size:         fh.Size,
		ContentType:  contentType,
	}, nil
}

// ResolvePath returns the on-disk path for (subfolder, name), or an error if
// path traversal is detected.
func (s *service) ResolvePath(subfolder, name string) (string, error) {
	sub := sanitize(subfolder)
	if strings.ContainsAny(name, "/\\") || name == "" || name == "." || name == ".." {
		return "", errors.New("invalid file name")
	}
	path := filepath.Join(s.basePath, sub, name)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(s.basePath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, base) {
		return "", errors.New("path traversal blocked")
	}
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func sanitize(subfolder string) string {
	subfolder = strings.TrimSpace(subfolder)
	if subfolder == "" {
		return ""
	}
	subfolder = strings.ReplaceAll(subfolder, "\\", "/")
	parts := strings.Split(subfolder, "/")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			continue
		}
		clean = append(clean, p)
	}
	return strings.Join(clean, "/")
}
