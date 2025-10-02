package controllers

import (
	_ "embed"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/input"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases/output"
)

const (
	userIDCookieName   = "waiting_room_user_id"
	defaultWaitingPath = "/waiting-room"
)

//go:embed templates/waiting_room.html
var waitingPageHTML string

var waitingPageTemplate = template.Must(template.New("waiting-page").Parse(waitingPageHTML))

// ブラウザアクセス時に待機室判定を行い、入場可能ならターゲットへ、待機中なら待機ページへリダイレクト
type EnterWaitingRoomWebController struct {
	useCase     *usecases.EnterWaitingRoomUseCase
	targetURL   string
	waitingPath string
}

func NewEnterWaitingRoomWebController(useCase *usecases.EnterWaitingRoomUseCase, targetURL string, waitingPath string) *EnterWaitingRoomWebController {
	if waitingPath == "" {
		waitingPath = defaultWaitingPath
	}
	return &EnterWaitingRoomWebController{
		useCase:     useCase,
		targetURL:   targetURL,
		waitingPath: waitingPath,
	}
}

func (c *EnterWaitingRoomWebController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := ensureUserIDCookie(w, r)
	if err != nil {
		http.Error(w, "failed to resolve user", http.StatusInternalServerError)
		return
	}

	in := input.EnterWaitingRoomInput{
		UserID: userID,
		Now:    time.Now(),
	}

	out, execErr := c.useCase.Execute(r.Context(), in)
	if execErr != nil {
		http.Error(w, execErr.Error(), http.StatusInternalServerError)
		return
	}

	if out.Outcome == output.EnterWaitingRoomOutcomeEnterTarget {
		http.Redirect(w, r, c.targetURL, http.StatusFound)
		return
	}

	waitURL := buildWaitingURL(c.waitingPath, out)
	http.Redirect(w, r, waitURL, http.StatusFound)
}

func ensureUserIDCookie(w http.ResponseWriter, r *http.Request) (uuid.UUID, error) {
	if cookie, err := r.Cookie(userIDCookieName); err == nil {
		if id, parseErr := uuid.Parse(cookie.Value); parseErr == nil {
			return id, nil
		}
	}

	newID := uuid.New()
	cookie := &http.Cookie{
		Name:     userIDCookieName,
		Value:    newID.String(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
	}
	http.SetCookie(w, cookie)
	return newID, nil
}

func buildWaitingURL(waitingPath string, out output.EnterWaitingRoomOutput) string {
	u, _ := url.Parse(waitingPath)
	q := u.Query()
	if out.TicketID != nil {
		q.Set("ticketId", out.TicketID.String())
	}
	if out.NewlyIssuedTicket {
		q.Set("newTicket", "1")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// 待機中ユーザー向けの簡易ページを返すハンドラを生成
func NewWaitingRoomPageHandler(refreshInterval time.Duration) http.Handler {
	refreshSeconds := int(refreshInterval.Seconds())
	if refreshSeconds <= 0 {
		refreshSeconds = 5
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		view := waitingPageViewModel{
			RefreshSeconds: refreshSeconds,
		}

		if ticketID := r.URL.Query().Get("ticketId"); ticketID != "" {
			if _, err := uuid.Parse(ticketID); err == nil {
				view.TicketID = ticketID
				view.HasTicket = true
			}
		}
		if r.URL.Query().Get("newTicket") == "1" {
			view.NewlyIssuedTicket = true
		}

		if err := waitingPageTemplate.Execute(w, view); err != nil {
			var templateErr *template.Error
			if errors.As(err, &templateErr) {
				http.Error(w, templateErr.Error(), http.StatusInternalServerError)
				return
			}
			http.Error(w, "failed to render", http.StatusInternalServerError)
		}
	})
}

type waitingPageViewModel struct {
	RefreshSeconds    int
	HasTicket         bool
	TicketID          string
	NewlyIssuedTicket bool
}
