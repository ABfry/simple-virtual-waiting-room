package services

import (
	"time"

	"github.com/google/uuid"

	domainentities "github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
)

// セッション管理の最小実装。
type SimpleSessionPolicy struct{}

func NewSimpleSessionPolicy() *SimpleSessionPolicy {
	return &SimpleSessionPolicy{}
}

func (SimpleSessionPolicy) Create(userID uuid.UUID, startedAt time.Time) domainentities.Session {
	return domainentities.NewSession(userID, startedAt)
}

func (SimpleSessionPolicy) End(session *domainentities.Session, reason domainentities.SessionEndReason, endedAt time.Time) {
	session.End(reason, endedAt)
}
