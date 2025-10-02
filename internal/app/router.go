package app

import (
	"net/http"

	"github.com/ABfry/simple-virtual-waiting-room/internal/middleware"
)

// NewRouter は HTTP ルーティングとミドルウェアを組み立てる。
func NewRouter(deps *Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/waiting-room/enter", deps.EnterWaitingRoomHandler)
	mux.Handle("/waiting-room", deps.WaitingRoomHandler)
	mux.Handle("/", deps.RootHandler)
	mux.HandleFunc("GET /healthz", healthHandler)

	return middleware.Logging(mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
