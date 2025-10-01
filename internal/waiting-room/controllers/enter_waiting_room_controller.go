package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/input"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/output"
)

type EnterWaitingRoomPresenter interface {
	Present(ctx context.Context, w http.ResponseWriter, out output.EnterWaitingRoomOutput, err error)
}

// 待機室入場 APIのエントリーポイント
type EnterWaitingRoomController struct {
	useCase   *usecases.EnterWaitingRoomUseCase
	presenter EnterWaitingRoomPresenter
}

func NewEnterWaitingRoomController(useCase *usecases.EnterWaitingRoomUseCase, presenter EnterWaitingRoomPresenter) *EnterWaitingRoomController {
	return &EnterWaitingRoomController{useCase: useCase, presenter: presenter}
}

func (c *EnterWaitingRoomController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		c.presenter.Present(r.Context(), w, output.EnterWaitingRoomOutput{}, methodNotAllowedError{method: r.Method})
		return
	}

	var payload enterWaitingRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		c.presenter.Present(r.Context(), w, output.EnterWaitingRoomOutput{}, invalidRequestError{msg: "failed to parse request body"}) // リクエストJSONのデコード失敗
		return
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		c.presenter.Present(r.Context(), w, output.EnterWaitingRoomOutput{}, invalidRequestError{msg: "userId must be a valid UUID"})
		return
	}

	var userName *string
	if trimmed := strings.TrimSpace(payload.UserName); trimmed != "" {
		userName = &trimmed
	}

	in := input.EnterWaitingRoomInput{
		UserID:   userID,
		UserName: userName,
		Now:      time.Now(),
	}

	out, execErr := c.useCase.Execute(r.Context(), in)
	// presenter側でレスポンス整形を行う
	c.presenter.Present(r.Context(), w, out, execErr)
}

type enterWaitingRoomRequest struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
}

type invalidRequestError struct {
	msg string
}

func (e invalidRequestError) Error() string {
	return e.msg
}

func (e invalidRequestError) StatusCode() int {
	return http.StatusBadRequest
}

func (e invalidRequestError) Code() string {
	return "invalid_request"
}

type methodNotAllowedError struct {
	method string
}

func (e methodNotAllowedError) Error() string {
	return "method " + e.method + " not allowed"
}

func (e methodNotAllowedError) StatusCode() int {
	return http.StatusMethodNotAllowed
}

func (e methodNotAllowedError) Code() string {
	return "method_not_allowed"
}
