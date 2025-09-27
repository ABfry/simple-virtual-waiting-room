package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

// セッション管理
type SessionPolicy interface {
	Create(userID uuid.UUID, startedAt time.Time) entities.Session
	End(session *entities.Session, reason entities.SessionEndReason, endedAt time.Time)
}
