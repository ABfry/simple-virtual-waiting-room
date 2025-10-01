package presenters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/repositories"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/output"
)

type EnterWaitingRoomPresenter struct{}

func NewEnterWaitingRoomPresenter() *EnterWaitingRoomPresenter {
	return &EnterWaitingRoomPresenter{}
}

// ユースケースの実行結果を基にレスポンスを構築
// エラーならpresentErrorに委譲
func (p *EnterWaitingRoomPresenter) Present(ctx context.Context, w http.ResponseWriter, out output.EnterWaitingRoomOutput, err error) {
	if err != nil {
		p.presentError(w, err)
		return
	}

	view := enterWaitingRoomViewModel{
		Outcome:            translateOutcome(out.Outcome),
		SessionID:          out.SessionID,
		TicketID:           out.TicketID,
		NewlyIssuedSession: out.NewlyIssuedSession,
		NewlyIssuedTicket:  out.NewlyIssuedTicket,
	}

	status := http.StatusOK
	if out.Outcome == output.EnterWaitingRoomOutcomeRedirectWaitingRoom {
		status = http.StatusAccepted
	}

	writeJSON(w, status, view)
}

// エラー種別に応じてステータスコード・エラーコードを決める
func (p *EnterWaitingRoomPresenter) presentError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"

	switch e := err.(type) {
	case statusError:
		status = e.StatusCode()
		code = e.Code()
	default:
		switch {
		case errors.Is(err, repositories.ErrNotFound):
			status = http.StatusNotFound
			code = "not_found"
		default:
			status = http.StatusInternalServerError
			code = "internal_error"
		}
	}

	writeJSON(w, status, errorResponse{Code: code, Message: err.Error()})
}

func translateOutcome(o output.EnterWaitingRoomOutcome) string {
	switch o {
	case output.EnterWaitingRoomOutcomeEnterTarget:
		return "enter_target"
	case output.EnterWaitingRoomOutcomeRedirectWaitingRoom:
		return "redirect_waiting_room"
	default:
		return "unknown"
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type enterWaitingRoomViewModel struct {
	Outcome            string     `json:"outcome"`
	SessionID          *uuid.UUID `json:"sessionId,omitempty"`
	TicketID           *uuid.UUID `json:"ticketId,omitempty"`
	NewlyIssuedSession bool       `json:"newlyIssuedSession"`
	NewlyIssuedTicket  bool       `json:"newlyIssuedTicket"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type statusError interface {
	error
	StatusCode() int
	Code() string
}
