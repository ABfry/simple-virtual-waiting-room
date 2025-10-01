package output

import "github.com/google/uuid"

// 待機室入場処理の結果
type EnterWaitingRoomOutcome int

const (
	EnterWaitingRoomOutcomeUnknown EnterWaitingRoomOutcome = iota
	EnterWaitingRoomOutcomeEnterTarget
	EnterWaitingRoomOutcomeRedirectWaitingRoom
)

type EnterWaitingRoomOutput struct {
	Outcome            EnterWaitingRoomOutcome
	SessionID          *uuid.UUID
	TicketID           *uuid.UUID
	NewlyIssuedSession bool
	NewlyIssuedTicket  bool
}
