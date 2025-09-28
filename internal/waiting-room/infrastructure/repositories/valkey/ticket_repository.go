package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/repositories"
)

type TicketRepository struct {
	client    redis.Cmdable
	namespace string
}

func NewTicketRepository(client redis.Cmdable, opts ...Option) *TicketRepository {
	cfg := applyOptions(opts)
	return &TicketRepository{client: client, namespace: cfg.namespace}
}

func (r *TicketRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Ticket, error) {
	raw, err := r.client.Get(ctx, r.key(id)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	var record ticketRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, err
	}

	ticket, err := record.toEntity()
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (r *TicketRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*entities.Ticket, error) {
	ticketIDStr, err := r.client.Get(ctx, r.activeKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	ticketID, err := uuid.Parse(ticketIDStr)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, ticketID)
}

func (r *TicketRepository) Save(ctx context.Context, ticket *entities.Ticket) error {
	if ticket == nil {
		return fmt.Errorf("ticket repository: ticket is nil")
	}

	record := newTicketRecord(ticket)
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	if err := r.client.Set(ctx, r.key(ticket.ID), payload, 0).Err(); err != nil {
		return err
	}

	if ticket.Status == entities.TicketStatusWaiting || ticket.Status == entities.TicketStatusAdmitted {
		if err := r.client.Set(ctx, r.activeKey(ticket.UserID), ticket.ID.String(), 0).Err(); err != nil {
			return err
		}
	} else {
		if err := r.client.Del(ctx, r.activeKey(ticket.UserID)).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (r *TicketRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ticket, err := r.GetByID(ctx, id)
	if err != nil {
		if !errors.Is(err, repositories.ErrNotFound) {
			return err
		}
	} else if ticket != nil {
		if err := r.client.Del(ctx, r.activeKey(ticket.UserID)).Err(); err != nil {
			return err
		}
	}

	if err := r.client.Del(ctx, r.key(id)).Err(); err != nil {
		return err
	}
	return nil
}

func (r *TicketRepository) key(id uuid.UUID) string {
	return fmt.Sprintf("%sticket:%s", r.namespace, id.String())
}

func (r *TicketRepository) activeKey(userID uuid.UUID) string {
	return fmt.Sprintf("%sticket:active:%s", r.namespace, userID.String())
}

type ticketRecord struct {
	ID        string                `json:"id"`
	UserID    string                `json:"user_id"`
	CreatedAt time.Time             `json:"created_at"`
	ExpiresAt time.Time             `json:"expires_at"`
	Status    entities.TicketStatus `json:"status"`
}

func newTicketRecord(ticket *entities.Ticket) ticketRecord {
	return ticketRecord{
		ID:        ticket.ID.String(),
		UserID:    ticket.UserID.String(),
		CreatedAt: ticket.CreatedAt,
		ExpiresAt: ticket.ExpiresAt,
		Status:    ticket.Status,
	}
}

func (r ticketRecord) toEntity() (*entities.Ticket, error) {
	ticketID, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return nil, err
	}

	ticket := entities.Ticket{
		ID:        ticketID,
		UserID:    userID,
		CreatedAt: r.CreatedAt,
		ExpiresAt: r.ExpiresAt,
		Status:    r.Status,
	}
	return &ticket, nil
}
