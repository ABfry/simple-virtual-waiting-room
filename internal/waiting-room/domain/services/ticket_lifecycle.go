package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

// チケットのライフサイクル管理
type TicketLifecycle interface {
	Issue(userID uuid.UUID, ttl time.Duration, now time.Time) entities.Ticket
	Validate(ticket entities.Ticket, now time.Time) error
	ValidateAndUse(ticket *entities.Ticket, now time.Time) error
	MarkWaiting(ticket *entities.Ticket)
	MarkAdmitted(ticket *entities.Ticket)
	MarkExpired(ticket *entities.Ticket)
}
