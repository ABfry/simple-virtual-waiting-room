package services

import (
	"time"

	"github.com/google/uuid"

	domainentities "github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
	domainservices "github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/services"
)

// チケット管理の最小実装。
type SimpleTicketLifecycle struct{}

func NewSimpleTicketLifecycle() *SimpleTicketLifecycle {
	return &SimpleTicketLifecycle{}
}

func (SimpleTicketLifecycle) Issue(userID uuid.UUID, ttl time.Duration, now time.Time) domainentities.Ticket {
	return domainentities.NewTicket(userID, ttl, now)
}

func (SimpleTicketLifecycle) Validate(ticket domainentities.Ticket, now time.Time) error {
	if ticket.IsExpired(now) {
		return domainservices.ErrTicketExpired
	}
	if ticket.Status == domainentities.TicketStatusUsed {
		return domainservices.ErrTicketAlreadyUsed
	}
	if ticket.Status == domainentities.TicketStatusExpired {
		return domainservices.ErrTicketExpired
	}
	if ticket.Status == domainentities.TicketStatusUnknown {
		return domainservices.ErrInvalidTicketStatus
	}
	return nil
}

func (l *SimpleTicketLifecycle) ValidateAndUse(ticket *domainentities.Ticket, now time.Time) error {
	if err := l.Validate(*ticket, now); err != nil {
		return err
	}
	if ticket.Status != domainentities.TicketStatusAdmitted {
		return domainservices.ErrInvalidTicketStatus
	}
	ticket.MarkUsed()
	return nil
}

func (SimpleTicketLifecycle) MarkWaiting(ticket *domainentities.Ticket) {
	ticket.MarkWaiting()
}

func (SimpleTicketLifecycle) MarkAdmitted(ticket *domainentities.Ticket) {
	ticket.MarkAdmitted()
}

func (SimpleTicketLifecycle) MarkExpired(ticket *domainentities.Ticket) {
	ticket.MarkExpired()
}
