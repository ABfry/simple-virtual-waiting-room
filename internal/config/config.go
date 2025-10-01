package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Config はアプリ全体の設定値を保持する。
type Config struct {
	HTTPAddr        string
	WaitingRoomID   uuid.UUID
	WaitingRoomCap  int
	TicketTTL       time.Duration
	ShutdownTimeout time.Duration
	ValkeyAddr      string
	ValkeyPassword  string
	ValkeyDB        int
	ValkeyNamespace string
}

// Load は環境変数から設定を読み込み、Config に詰める。
func Load() (Config, error) {
	var cfg Config

	cfg.HTTPAddr = getEnvWithDefault("HTTP_ADDR", ":8080")
	if !strings.HasPrefix(cfg.HTTPAddr, ":") && !strings.Contains(cfg.HTTPAddr, ":") {
		cfg.HTTPAddr = ":" + cfg.HTTPAddr
	}

	roomIDStr := getEnvWithDefault("WAITING_ROOM_ID", "11111111-1111-1111-1111-111111111111")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid WAITING_ROOM_ID: %w", err)
	}
	cfg.WaitingRoomID = roomID

	capacityStr := getEnvWithDefault("WAITING_ROOM_CAPACITY", "10")
	capacity, err := strconv.Atoi(capacityStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid WAITING_ROOM_CAPACITY: %w", err)
	}
	cfg.WaitingRoomCap = capacity

	ttlStr := getEnvWithDefault("WAITING_ROOM_TICKET_TTL", "5m")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid WAITING_ROOM_TICKET_TTL: %w", err)
	}
	cfg.TicketTTL = ttl

	shutdownStr := getEnvWithDefault("HTTP_SHUTDOWN_TIMEOUT", "10s")
	shutdown, err := time.ParseDuration(shutdownStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid HTTP_SHUTDOWN_TIMEOUT: %w", err)
	}
	if shutdown <= 0 {
		shutdown = 10 * time.Second
	}
	cfg.ShutdownTimeout = shutdown

	cfg.ValkeyAddr = getEnvWithDefault("VALKEY_ADDR", "localhost:6379")
	cfg.ValkeyPassword = os.Getenv("VALKEY_PASSWORD")
	valkeyDBStr := getEnvWithDefault("VALKEY_DB", "0")
	valkeyDB, err := strconv.Atoi(valkeyDBStr)
	if err != nil {
		return cfg, fmt.Errorf("invalid VALKEY_DB: %w", err)
	}
	cfg.ValkeyDB = valkeyDB
	cfg.ValkeyNamespace = getEnvWithDefault("VALKEY_NAMESPACE", "waiting-room")

	return cfg, nil
}

func getEnvWithDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
