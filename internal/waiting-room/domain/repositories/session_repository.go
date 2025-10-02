package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

type SessionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Session, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*entities.Session, error)
	Save(ctx context.Context, session *entities.Session) error
	CountActive(ctx context.Context) (int64, error)
	RefreshTTL(ctx context.Context, session *entities.Session, ttl time.Duration) error
}
