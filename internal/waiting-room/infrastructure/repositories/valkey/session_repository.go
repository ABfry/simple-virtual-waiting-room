package valkey

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/repositories"
)

type SessionRepository struct {
	client    redis.Cmdable
	namespace string
}

func NewSessionRepository(client redis.Cmdable, opts ...Option) *SessionRepository {
	cfg := applyOptions(opts)
	return &SessionRepository{client: client, namespace: cfg.namespace}
}

func (r *SessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Session, error) {
	raw, err := r.client.Get(ctx, r.key(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	var record sessionRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}

	session, err := record.toEntity()
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *SessionRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*entities.Session, error) {
	sessionIDStr, err := r.client.Get(ctx, r.activeKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, sessionID)
}

func (r *SessionRepository) Save(ctx context.Context, session *entities.Session) error {
	if session == nil {
		return fmt.Errorf("session repository: session is nil")
	}

	record := newSessionRecord(session)
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	if err := r.client.Set(ctx, r.key(session.ID), payload, 0).Err(); err != nil {
		return err
	}

	if session.IsActive() {
		if err := r.client.Set(ctx, r.activeKey(session.UserID), session.ID.String(), 0).Err(); err != nil {
			return err
		}
		if err := r.client.SAdd(ctx, r.activeSetKey(), session.ID.String()).Err(); err != nil {
			return err
		}
	} else {
		if err := r.client.Del(ctx, r.activeKey(session.UserID)).Err(); err != nil {
			return err
		}
		if err := r.client.SRem(ctx, r.activeSetKey(), session.ID.String()).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (r *SessionRepository) key(id uuid.UUID) string {
	return fmt.Sprintf("%ssession:%s", r.namespace, id.String())
}

func (r *SessionRepository) activeKey(userID uuid.UUID) string {
	return fmt.Sprintf("%ssession:active:%s", r.namespace, userID.String())
}

func (r *SessionRepository) activeSetKey() string {
	return fmt.Sprintf("%ssession:active", r.namespace)
}

func (r *SessionRepository) CountActive(ctx context.Context) (int64, error) {
	members, err := r.client.SMembers(ctx, r.activeSetKey()).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	var count int64
	for _, member := range members {
		sessionID, parseErr := uuid.Parse(member)
		if parseErr != nil {
			if remErr := r.client.SRem(ctx, r.activeSetKey(), member).Err(); remErr != nil {
				return 0, remErr
			}
			continue
		}
		exists, err := r.client.Exists(ctx, r.key(sessionID)).Result()
		if err != nil {
			return 0, err
		}
		if exists == 0 {
			if err := r.client.SRem(ctx, r.activeSetKey(), member).Err(); err != nil {
				return 0, err
			}
			continue
		}
		count++
	}
	return count, nil
}

func (r *SessionRepository) RefreshTTL(ctx context.Context, session *entities.Session, ttl time.Duration) error {
	if ttl <= 0 || session == nil {
		return nil
	}
	if err := r.client.Expire(ctx, r.key(session.ID), ttl).Err(); err != nil {
		return err
	}
	if err := r.client.Expire(ctx, r.activeKey(session.UserID), ttl).Err(); err != nil {
		return err
	}
	if err := r.client.SAdd(ctx, r.activeSetKey(), session.ID.String()).Err(); err != nil {
		return err
	}
	return nil
}

type sessionRecord struct {
	ID        string                    `json:"id"`
	UserID    string                    `json:"user_id"`
	StartedAt time.Time                 `json:"started_at"`
	EndedAt   *time.Time                `json:"ended_at,omitempty"`
	Status    entities.SessionStatus    `json:"status"`
	Reason    entities.SessionEndReason `json:"reason"`
}

func newSessionRecord(session *entities.Session) sessionRecord {
	return sessionRecord{
		ID:        session.ID.String(),
		UserID:    session.UserID.String(),
		StartedAt: session.StartedAt,
		EndedAt:   cloneTimePtr(session.EndedAt),
		Status:    session.Status,
		Reason:    session.Reason,
	}
}

func (r sessionRecord) toEntity() (*entities.Session, error) {
	sessionID, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return nil, err
	}

	session := entities.Session{
		ID:        sessionID,
		UserID:    userID,
		StartedAt: r.StartedAt,
		Status:    r.Status,
		Reason:    r.Reason,
	}
	session.EndedAt = cloneTimePtr(r.EndedAt)
	return &session, nil
}
