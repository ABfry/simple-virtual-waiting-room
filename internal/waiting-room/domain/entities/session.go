package entities

import (
	"time"

	"github.com/google/uuid"
)

//go:generate stringer -type=SessionStatus -trimprefix=SessionStatus
type SessionStatus int

const (
	SessionStatusUnknown SessionStatus = iota
	SessionStatusActive
	SessionStatusTimeout
	SessionStatusClosed
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	StartedAt time.Time
	ExitedAt  *time.Time
	Status    SessionStatus
}

func NewSession(userID uuid.UUID, startedAt time.Time) Session {
	return Session{
		ID:        uuid.New(),
		UserID:    userID,
		StartedAt: startedAt,
		Status:    SessionStatusActive,
	}
}

func (s *Session) Close(at time.Time) {
	if at.IsZero() {
		s.ExitedAt = nil
	} else {
		exited := at
		s.ExitedAt = &exited
	}
	s.Status = SessionStatusClosed
}

func (s *Session) Timeout(at time.Time) {
	if at.IsZero() {
		s.ExitedAt = nil
	} else {
		exited := at
		s.ExitedAt = &exited
	}
	s.Status = SessionStatusTimeout
}

func (s Session) IsActive() bool {
	return s.Status == SessionStatusActive
}
