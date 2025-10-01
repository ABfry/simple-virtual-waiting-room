package input

import (
	"time"

	"github.com/google/uuid"
)

type EnterWaitingRoomInput struct {
	UserID   uuid.UUID
	UserName *string
	Now      time.Time
}

func (i EnterWaitingRoomInput) EffectiveTime() time.Time {
	if i.Now.IsZero() {
		return time.Now()
	}
	return i.Now
}
