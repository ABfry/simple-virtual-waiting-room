package controllers

import (
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

// EnterWaitingRoomWebController はブラウザアクセス時に待機室判定を行い、
// 入場可能ならターゲットへ、待機中なら待機ページへリダイレクトさせる。
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

// NewWaitingRoomPageHandler は待機中ユーザー向けの簡易ページを返すハンドラを生成する。
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

var waitingPageTemplate = template.Must(template.New("waiting-page").Parse(`<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="utf-8">
<title>待機室</title>
<meta http-equiv="refresh" content="{{.RefreshSeconds}};url=/">
<style>
body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    background-color: #f4f4f5;
    color: #1f2933;
    margin: 0;
    padding: 0;
}
main {
    max-width: 480px;
    margin: 80px auto;
    background: #ffffff;
    border-radius: 12px;
    box-shadow: 0 10px 25px rgba(15, 23, 42, 0.08);
    padding: 32px;
    text-align: center;
}
h1 {
    margin-top: 0;
    font-size: 1.8rem;
}
p {
    line-height: 1.6;
}
.ticket {
    margin: 24px 0 8px;
    padding: 12px;
    background: #eef2ff;
    border-radius: 8px;
    font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
}
a.button {
    display: inline-block;
    margin-top: 16px;
    padding: 10px 16px;
    background: #2563eb;
    color: #ffffff;
    text-decoration: none;
    border-radius: 6px;
}
a.button:hover {
    background: #1d4ed8;
}
</style>
</head>
<body>
<main>
    <h1>ただいま入場待ちです</h1>
    {{if .NewlyIssuedTicket}}
    <p>待機チケットを発行しました。順番が来るまでしばらくお待ちください。</p>
    {{else}}
    <p>順番が来るまでお待ちください。ページは自動的に更新されます。</p>
    {{end}}
    {{if .HasTicket}}
    <p class="ticket">チケットID: {{.TicketID}}</p>
    {{end}}
    <p>自動的に <strong>{{.RefreshSeconds}} 秒後</strong> に状態を確認します。</p>
    <a class="button" href="/">今すぐ状況を確認する</a>
</main>
</body>
</html>`))
