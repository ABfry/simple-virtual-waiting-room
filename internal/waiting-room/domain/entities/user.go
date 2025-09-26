package entities

import (
	"time"

	"github.com/google/uuid"
)

//go:generate stringer -type=UserStatus -trimprefix=UserStatus
type UserStatus int

const (
	UserStatusUnknown UserStatus = iota
	UserStatusWaiting
	UserStatusEntered
	UserStatusExited
)

type User struct {
	ID        uuid.UUID
	Name      *string
	Status    UserStatus
	QueuedAt  time.Time
	EnteredAt *time.Time
	ExitedAt  *time.Time
}

func NewQueuedUser(id uuid.UUID, name *string, queuedAt time.Time) User {
	return User{
		ID:       id,
		Name:     name,
		Status:   UserStatusWaiting,
		QueuedAt: queuedAt,
	}
}

func (u *User) MarkEntered(at time.Time) {
	u.Status = UserStatusEntered
	if at.IsZero() {
		u.EnteredAt = nil
		return
	}
	// ensure we retain a copy
	entered := at
	u.EnteredAt = &entered
}

func (u *User) MarkExited(at time.Time) {
	u.Status = UserStatusExited
	if at.IsZero() {
		u.ExitedAt = nil
		return
	}
	exited := at
	u.ExitedAt = &exited
}

func (u *User) ResetToWaiting(requeuedAt time.Time) {
	u.Status = UserStatusWaiting
	u.EnteredAt = nil
	u.ExitedAt = nil
	if !requeuedAt.IsZero() {
		u.QueuedAt = requeuedAt
	}
}

func (u User) HasEntered() bool {
	return u.Status == UserStatusEntered && u.EnteredAt != nil
}

func (u User) HasExited() bool {
	return u.Status == UserStatusExited && u.ExitedAt != nil
}
