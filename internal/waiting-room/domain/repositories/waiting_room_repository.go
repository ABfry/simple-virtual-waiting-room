package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

type WaitingRoomRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.WaitingRoom, error)
	Save(ctx context.Context, room *entities.WaitingRoom) error
}
