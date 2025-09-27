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

type SessionEndReason int

const (
	SessionEndReasonUnknown SessionEndReason = iota
	SessionEndReasonUserExit
	SessionEndReasonTimeout
	SessionEndReasonForceClosed
)

func (r SessionEndReason) String() string {
	switch r {
	case SessionEndReasonUserExit:
		return "user_exit"
	case SessionEndReasonTimeout:
		return "timeout"
	case SessionEndReasonForceClosed:
		return "force_closed"
	default:
		return "unknown"
	}
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	StartedAt time.Time
	EndedAt   *time.Time
	Status    SessionStatus
	Reason    SessionEndReason
}

func NewSession(userID uuid.UUID, startedAt time.Time) Session {
	return Session{
		ID:        uuid.New(),
		UserID:    userID,
		StartedAt: startedAt,
		Status:    SessionStatusActive,
		Reason:    SessionEndReasonUnknown,
	}
}

func (s *Session) End(reason SessionEndReason, at time.Time) {
	if at.IsZero() {
		s.EndedAt = nil
	} else {
		ended := at
		s.EndedAt = &ended
	}
	s.Reason = reason
	s.Status = mapSessionStatusFromReason(reason)
}

func mapSessionStatusFromReason(reason SessionEndReason) SessionStatus {
	switch reason {
	case SessionEndReasonTimeout:
		return SessionStatusTimeout
	case SessionEndReasonUserExit, SessionEndReasonForceClosed:
		return SessionStatusClosed
	default:
		return SessionStatusUnknown
	}
}

func (s Session) IsActive() bool {
	return s.Status == SessionStatusActive
}
