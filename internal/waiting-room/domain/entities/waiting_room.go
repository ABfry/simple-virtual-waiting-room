package entities

import (
	"time"

	"github.com/google/uuid"
)

type WaitingRoom struct {
	ID       uuid.UUID
	Capacity int
	TTL      time.Duration
	Queue    []uuid.UUID
}

func NewWaitingRoom(id uuid.UUID, capacity int, ttl time.Duration) WaitingRoom {
	return WaitingRoom{
		ID:       id,
		Capacity: capacity,
		TTL:      ttl,
		Queue:    make([]uuid.UUID, 0, capacity),
	}
}

func (w *WaitingRoom) Enqueue(ticketID uuid.UUID) {
	w.Queue = append(w.Queue, ticketID)
}

func (w *WaitingRoom) Dequeue(count int) []uuid.UUID {
	if count <= 0 || len(w.Queue) == 0 {
		return nil
	}
	if count > len(w.Queue) {
		count = len(w.Queue)
	}
	batch := append([]uuid.UUID(nil), w.Queue[:count]...)
	w.Queue = append([]uuid.UUID(nil), w.Queue[count:]...)
	return batch
}

func (w *WaitingRoom) Remove(ticketID uuid.UUID) bool {
	for idx, id := range w.Queue {
		if id == ticketID {
			w.Queue = append(w.Queue[:idx], w.Queue[idx+1:]...)
			return true
		}
	}
	return false
}

func (w WaitingRoom) Len() int {
	return len(w.Queue)
}

func (w WaitingRoom) HasCapacity() bool {
	if w.Capacity <= 0 {
		return true
	}
	return len(w.Queue) < w.Capacity
}
