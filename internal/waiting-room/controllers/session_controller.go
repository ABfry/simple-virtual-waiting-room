package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/repositories"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases"
)

// SessionHeartbeatController はセッションの生存確認リクエストを処理する。
type SessionHeartbeatController struct {
	useCase *usecases.EnterWaitingRoomUseCase
}

func NewSessionHeartbeatController(useCase *usecases.EnterWaitingRoomUseCase) *SessionHeartbeatController {
	return &SessionHeartbeatController{useCase: useCase}
}

func (c *SessionHeartbeatController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload sessionHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "failed to parse request body", http.StatusBadRequest)
		return
	}

	sessionID, err := uuid.Parse(payload.SessionID)
	if err != nil {
		http.Error(w, "sessionId must be a valid UUID", http.StatusBadRequest)
		return
	}

	if err := c.useCase.KeepSessionAlive(r.Context(), sessionID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type sessionHeartbeatRequest struct {
	SessionID string `json:"sessionId"`
}
