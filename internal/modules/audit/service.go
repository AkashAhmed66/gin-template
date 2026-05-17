package audit

import (
	"context"
	"sync"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service is the audit log API: a buffered async writer + a paginated reader
// for the admin-facing endpoint.
type Service interface {
	Record(log Log) // non-blocking
	Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Log], error)
	Wait(ctx context.Context)
}

// dbService writes through a buffered channel + a single worker goroutine.
// 1k buffer is enough to absorb bursts; under sustained pressure the channel
// blocks momentarily, which we prefer to dropping audit rows.
type dbService struct {
	db   *gorm.DB
	log  *zap.Logger
	ch   chan Log
	done chan struct{}
	wg   sync.WaitGroup
}

// NewService starts the background writer. Stop draining by calling Wait
// during shutdown.
func NewService(db *gorm.DB, log *zap.Logger) Service {
	s := &dbService{
		db:   db,
		log:  log,
		ch:   make(chan Log, 1024),
		done: make(chan struct{}),
	}
	s.wg.Add(1)
	go s.worker()
	return s
}

func (s *dbService) worker() {
	defer s.wg.Done()
	for entry := range s.ch {
		if err := s.db.Create(&entry).Error; err != nil {
			s.log.Warn("audit write failed",
				zap.String("path", entry.Path),
				zap.Error(err))
		}
	}
}

func (s *dbService) Record(entry Log) {
	select {
	case s.ch <- entry:
	default:
		// Buffer full — drop with a warn rather than blocking the request thread.
		s.log.Warn("audit buffer full — dropping record",
			zap.String("path", entry.Path),
			zap.String("request_id", entry.RequestID),
		)
	}
}

func (s *dbService) Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Log], error) {
	var (
		rows  []Log
		total int64
	)
	q := s.db.WithContext(ctx).Model(&Log{})
	if f.Username != "" {
		q = q.Where("username = ?", f.Username)
	}
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.Method != "" {
		q = q.Where("method = ?", f.Method)
	}
	if f.Path != "" {
		q = q.Where("path LIKE ?", "%"+f.Path+"%")
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}
	if f.ResourceID != "" {
		q = q.Where("resource_id = ?", f.ResourceID)
	}
	if f.StatusCode != nil {
		q = q.Where("status_code = ?", *f.StatusCode)
	}
	if f.RequestID != "" {
		q = q.Where("request_id = ?", f.RequestID)
	}
	if f.From != nil {
		q = q.Where("timestamp >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("timestamp <= ?", *f.To)
	}
	if err := q.Count(&total).Error; err != nil {
		return dto.PageResponse[Log]{}, err
	}
	if err := q.Order("timestamp DESC").
		Offset(page.Offset()).Limit(page.Limit()).
		Find(&rows).Error; err != nil {
		return dto.PageResponse[Log]{}, err
	}
	return dto.NewPage(rows, page, total), nil
}

func (s *dbService) Wait(ctx context.Context) {
	close(s.ch)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
