package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

// 入室ポリシー
type AdmissionPolicy interface {
	CanJoin(room *entities.WaitingRoom, user *entities.User, now time.Time) error
	PickNext(room *entities.WaitingRoom, limit int, now time.Time) ([]uuid.UUID, error)
}
