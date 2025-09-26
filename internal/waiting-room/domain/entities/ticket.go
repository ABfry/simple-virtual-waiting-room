package entities

import (
	"time"

	"github.com/google/uuid"
)

//go:generate stringer -type=TicketStatus -trimprefix=TicketStatus
type TicketStatus int

const (
	TicketStatusUnknown TicketStatus = iota
	TicketStatusValid
	TicketStatusUsed
	TicketStatusExpired
)

type Ticket struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time // チケットの有効期限
	Status    TicketStatus
}

func NewTicket(userID uuid.UUID, lifetime time.Duration, issuedAt time.Time) Ticket {
	return Ticket{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: issuedAt,
		ExpiresAt: issuedAt.Add(lifetime),
		Status:    TicketStatusValid,
	}
}

func (t Ticket) IsExpired(reference time.Time) bool {
	return !t.ExpiresAt.IsZero() && (reference.After(t.ExpiresAt) || reference.Equal(t.ExpiresAt))
}

func (t *Ticket) MarkUsed() {
	t.Status = TicketStatusUsed
}

func (t *Ticket) MarkExpired() {
	t.Status = TicketStatusExpired
}
