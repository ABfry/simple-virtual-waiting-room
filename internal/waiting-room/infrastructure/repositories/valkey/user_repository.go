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

type UserRepository struct {
	client    redis.Cmdable
	namespace string
}

func NewUserRepository(client redis.Cmdable, opts ...Option) *UserRepository {
	cfg := applyOptions(opts)
	return &UserRepository{client: client, namespace: cfg.namespace}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	raw, err := r.client.Get(ctx, r.key(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	var record userRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}

	user, err := record.toEntity()
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Save(ctx context.Context, user *entities.User) error {
	if user == nil {
		return fmt.Errorf("user repository: user is nil")
	}

	record := newUserRecord(user)
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	if err := r.client.Set(ctx, r.key(user.ID), payload, 0).Err(); err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) key(id uuid.UUID) string {
	return fmt.Sprintf("%suser:%s", r.namespace, id.String())
}

type userRecord struct {
	ID        string              `json:"id"`
	Name      *string             `json:"name,omitempty"`
	Status    entities.UserStatus `json:"status"`
	QueuedAt  time.Time           `json:"queued_at"`
	EnteredAt *time.Time          `json:"entered_at,omitempty"`
	ExitedAt  *time.Time          `json:"exited_at,omitempty"`
}

func newUserRecord(user *entities.User) userRecord {
	return userRecord{
		ID:        user.ID.String(),
		Name:      user.Name,
		Status:    user.Status,
		QueuedAt:  user.QueuedAt,
		EnteredAt: cloneTimePtr(user.EnteredAt),
		ExitedAt:  cloneTimePtr(user.ExitedAt),
	}
}

func (r userRecord) toEntity() (*entities.User, error) {
	userID, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, err
	}

	user := entities.User{
		ID:       userID,
		Name:     r.Name,
		Status:   r.Status,
		QueuedAt: r.QueuedAt,
	}
	user.EnteredAt = cloneTimePtr(r.EnteredAt)
	user.ExitedAt = cloneTimePtr(r.ExitedAt)
	return &user, nil
}
