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

type WaitingRoomRepository struct {
	client    redis.Cmdable
	namespace string
}

func NewWaitingRoomRepository(client redis.Cmdable, opts ...Option) *WaitingRoomRepository {
	cfg := applyOptions(opts)
	return &WaitingRoomRepository{
		client:    client,
		namespace: cfg.namespace,
	}
}

func (r *WaitingRoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.WaitingRoom, error) {
	key := r.key(id)
	raw, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	var record waitingRoomRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}

	room, err := record.toEntity()
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *WaitingRoomRepository) Save(ctx context.Context, room *entities.WaitingRoom) error {
	if room == nil {
		return fmt.Errorf("waiting room repository: room is nil")
	}

	record := newWaitingRoomRecord(room)
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	key := r.key(room.ID)
	if err := r.client.Set(ctx, key, payload, 0).Err(); err != nil {
		return err
	}
	return nil
}

func (r *WaitingRoomRepository) key(id uuid.UUID) string {
	return fmt.Sprintf("%swaiting_room:%s", r.namespace, id.String())
}

type waitingRoomRecord struct {
	ID         string   `json:"id"`
	Capacity   int      `json:"capacity"`
	TTLSeconds int64    `json:"ttl_seconds"`
	Queue      []string `json:"queue"`
}

func newWaitingRoomRecord(room *entities.WaitingRoom) waitingRoomRecord {
	queue := make([]string, len(room.Queue))
	for i, id := range room.Queue {
		queue[i] = id.String()
	}

	return waitingRoomRecord{
		ID:         room.ID.String(),
		Capacity:   room.Capacity,
		TTLSeconds: int64(room.TTL / time.Second),
		Queue:      queue,
	}
}

func (r waitingRoomRecord) toEntity() (*entities.WaitingRoom, error) {
	roomID, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, err
	}

	queue := make([]uuid.UUID, len(r.Queue))
	for i, raw := range r.Queue {
		uid, err := uuid.Parse(raw)
		if err != nil {
			return nil, err
		}
		queue[i] = uid
	}

	room := entities.WaitingRoom{
		ID:       roomID,
		Capacity: r.Capacity,
		TTL:      time.Duration(r.TTLSeconds) * time.Second,
		Queue:    queue,
	}
	return &room, nil
}
