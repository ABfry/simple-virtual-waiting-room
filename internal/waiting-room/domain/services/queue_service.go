package services

import (
	"github.com/google/uuid"
)

// 待機列操作
type QueueService interface {
	Enqueue(roomID uuid.UUID, userID uuid.UUID) error
	Remove(roomID uuid.UUID, userID uuid.UUID) (bool, error)
	Length(roomID uuid.UUID) (int, error)
	DequeueNextN(roomID uuid.UUID, n int) ([]uuid.UUID, error)
}
