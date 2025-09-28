package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

type TicketRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Ticket, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*entities.Ticket, error)
	Save(ctx context.Context, ticket *entities.Ticket) error
	Delete(ctx context.Context, id uuid.UUID) error
}
