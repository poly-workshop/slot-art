package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/lmittmann/tint"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	slotart "github.com/poly-workshop/slot-art"
	pb "github.com/poly-workshop/slot-art/gen/go/slot/v1"
	"github.com/poly-workshop/slot-art/internal/config"
	slotMiddleware "github.com/poly-workshop/slot-art/internal/middleware"
	"github.com/poly-workshop/slot-art/internal/provider"
	apiservice "github.com/poly-workshop/slot-art/internal/service"
	"github.com/poly-workshop/slot-art/internal/store"
	"github.com/poly-workshop/slot-art/internal/task"
)

func main() {
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: slogLevel(cfg.Log.Level),
	}))

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", "error", err)
		cancel()
		os.Exit(1)
	}
	cancel()

	st := store.New(rdb)

	enqueuer := task.NewEnqueuer(cfg.Asynq.RedisAddr, cfg.Asynq.RedisPassword)

	provider.Register(provider.NewOpenAI(provider.OpenAIConfig{
		APIKey:   cfg.OpenAIImage2.APIKey,
		Endpoint: cfg.OpenAIImage2.Endpoint,
		Model:    cfg.OpenAIImage2.Model,
	}))

	deps := apiservice.Deps{Cfg: cfg, Store: st, Enqueuer: enqueuer, Log: logger}
	slotSvc := apiservice.NewSlotArtService(deps)
	adminSvc := apiservice.NewAdminService(deps)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(apiservice.AdminUnaryInterceptor(cfg.Admin.Token)))
	pb.RegisterSlotArtServiceServer(grpcServer, slotSvc)
	pb.RegisterAdminServiceServer(grpcServer, adminSvc)
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	gatewayMux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(gatewayHeaderMatcher))
	if err := pb.RegisterSlotArtServiceHandlerServer(context.Background(), gatewayMux, slotSvc); err != nil {
		logger.Error("register slot gateway", "error", err)
		os.Exit(1)
	}
	if err := pb.RegisterAdminServiceHandlerServer(context.Background(), gatewayMux, adminSvc); err != nil {
		logger.Error("register admin gateway", "error", err)
		os.Exit(1)
	}

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "redis": "connected"})
	})
	httpMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Client().Ping(r.Context()).Err(); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "redis": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	httpMux.Handle("/api/admin/v1/", apiservice.AdminHTTPMiddleware(cfg.Admin.Token)(gatewayMux))
	httpMux.Handle("/api/v1/", gatewayMux)
	httpMux.Handle("/", slotart.StaticHandler())

	handler := grpcOrHTTP(grpcServer, httpMux)
	handler = slotMiddleware.RequestLogger(logger)(handler)
	handler = slotMiddleware.RateLimiter(rdb, slotMiddleware.DefaultRateLimitConfig())(handler)

	worker := task.NewWorker(cfg, logger)
	go func() {
		if err := worker.Start(st, enqueuer); err != nil {
			logger.Error("worker error", "error", err)
		}
	}()

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Info("starting server", "addr", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	shutdownCtx, shutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdown()

	worker.Stop()
	enqueuer.Close()
	grpcServer.GracefulStop()
	srv.Shutdown(shutdownCtx)
}

func gatewayHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "x-slot-uid", "x-slot-fingerprint", "authorization", "x-admin-token":
		return strings.ToLower(key), true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func grpcOrHTTP(grpcServer *grpc.Server, next http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}), &http2.Server{})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("write json", "error", err)
	}
}

func slogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
