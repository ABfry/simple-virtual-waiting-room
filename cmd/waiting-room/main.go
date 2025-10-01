package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ABfry/simple-virtual-waiting-room/internal/app"
	"github.com/ABfry/simple-virtual-waiting-room/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	deps, err := app.NewDependencies(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer deps.Close()

	router := app.NewRouter(deps)
	server := app.NewServer(cfg.HTTPAddr, router)

	if err := server.Run(ctx, cfg.ShutdownTimeout); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
