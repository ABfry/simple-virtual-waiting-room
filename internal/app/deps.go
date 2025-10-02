package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ABfry/simple-virtual-waiting-room/internal/config"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/controllers"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/domain/entities"
	valkeyrepo "github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/infrastructure/repositories/valkey"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/presenters"
	waitingroomservices "github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/services"
	"github.com/ABfry/simple-virtual-waiting-room/internal/waiting-room/usecases"
)

// Dependencies はアプリの依存関係を束ねる。
type Dependencies struct {
	RootHandler             http.Handler
	WaitingRoomHandler      http.Handler
	EnterWaitingRoomHandler http.Handler
	SessionHeartbeatHandler http.Handler
	redisClient             *redis.Client
}

// NewDependencies は環境に応じた依存を構築する。
func NewDependencies(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
		DB:       cfg.ValkeyDB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("connect valkey: %w", err)
	}

	namespaceOpt := valkeyrepo.WithNamespace(cfg.ValkeyNamespace)

	waitingRoomRepo := valkeyrepo.NewWaitingRoomRepository(redisClient, namespaceOpt)
	userRepo := valkeyrepo.NewUserRepository(redisClient, namespaceOpt)
	ticketRepo := valkeyrepo.NewTicketRepository(redisClient, namespaceOpt)
	sessionRepo := valkeyrepo.NewSessionRepository(redisClient, namespaceOpt)
	ticketLifecycle := waitingroomservices.NewSimpleTicketLifecycle()
	sessionPolicy := waitingroomservices.NewSimpleSessionPolicy()

	room := entities.NewWaitingRoom(cfg.WaitingRoomID, cfg.WaitingRoomCap, cfg.TicketTTL)
	if err := waitingRoomRepo.Save(ctx, &room); err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("seed waiting room: %w", err)
	}

	enterUseCase := usecases.NewEnterWaitingRoomUseCase(
		cfg.WaitingRoomID,
		waitingRoomRepo,
		userRepo,
		sessionRepo,
		ticketRepo,
		ticketLifecycle,
		sessionPolicy,
		nil,
		cfg.TargetMaxActiveSessions,
		cfg.SessionTTL,
	)

	presenter := presenters.NewEnterWaitingRoomPresenter()
	controller := controllers.NewEnterWaitingRoomController(enterUseCase, presenter)
	rootController := controllers.NewEnterWaitingRoomWebController(enterUseCase, cfg.TargetURL, "")
	heartbeatController := controllers.NewSessionHeartbeatController(enterUseCase)
	waitingPageHandler := controllers.NewWaitingRoomPageHandler(5 * time.Second)

	return &Dependencies{
		RootHandler:             rootController,
		WaitingRoomHandler:      waitingPageHandler,
		EnterWaitingRoomHandler: controller,
		SessionHeartbeatHandler: heartbeatController,
		redisClient:             redisClient,
	}, nil
}

// Close は依存終了処理を行う（今は no-op）。
func (d *Dependencies) Close() {
	if d.redisClient != nil {
		_ = d.redisClient.Close()
	}
}
