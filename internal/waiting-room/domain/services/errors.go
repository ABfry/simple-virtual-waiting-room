package services

import (
	"errors"
)

// エラー定義
var (
	ErrQueueFull           = errors.New("waiting room capacity exceeded")
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrTicketExpired       = errors.New("ticket expired")
	ErrTicketAlreadyUsed   = errors.New("ticket already used")
	ErrUserNotQueued       = errors.New("user not queued")
	ErrInvalidWaitingRoom  = errors.New("invalid waiting room")
	ErrInvalidTicketStatus = errors.New("invalid ticket status")
)
