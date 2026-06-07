package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/wemall/api-gateway/internal/clients"
	"github.com/wemall/api-gateway/internal/config"
	"github.com/wemall/api-gateway/internal/graph/generated"
	"github.com/wemall/api-gateway/internal/graph/gqlerrors"
	"github.com/wemall/api-gateway/internal/graph/model"
	"github.com/wemall/api-gateway/internal/graph/resolver"
	"github.com/wemall/api-gateway/internal/middleware"
	"github.com/wemall/pkg/logger"
)

var log = logger.New("api-gateway", "development")

func main() {
	cfg := config.Load()

	// ── gRPC clients ──────────────────────────────────────────────────────────
	grpcClients, err := clients.New(
		cfg.UserServiceAddr,
		cfg.ProductServiceAddr,
		cfg.OrderServiceAddr,
		cfg.SellerServiceAddr,
		cfg.InventoryServiceAddr,
		cfg.NotificationServiceAddr,
		cfg.ReviewServiceAddr,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to downstream services")
	}
	defer grpcClients.Close()

	// ── GraphQL handler ───────────────────────────────────────────────────────
	root := &resolver.Resolver{Clients: grpcClients}

	gqlCfg := generated.Config{
		Resolvers: root,
		Directives: generated.DirectiveRoot{
			HasRole: hasRoleDirective(cfg.JWTSecret),
		},
	}

	gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(gqlCfg))
	gqlHandler.SetErrorPresenter(gqlerrors.ErrorPresenter)
	gqlHandler.SetRecoverFunc(gqlerrors.RecoverFunc)
	authMW := middleware.Auth(cfg.JWTSecret)

	// ── HTTP mux ──────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP"}`))
	})

	// GraphQL Playground (GET /playground)
	mux.Handle("/playground", playground.Handler("WeMall GraphQL", "/graphql"))

	// GraphQL endpoint — wrapped with JWT auth middleware
	mux.Handle("/graphql", authMW(gqlHandler))

	// ── HTTP server ────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Msgf("API Gateway on :%s  |  playground: http://localhost:%s/playground", cfg.Port, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// ── @hasRole directive ────────────────────────────────────────────────────────

func hasRoleDirective(jwtSecret string) func(ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {
		userRole := middleware.RoleFromCtx(ctx)
		_, authenticated := middleware.UserIDFromCtx(ctx)
		if !authenticated {
			return nil, gqlerrors.Unauthenticated("authentication required")
		}
		switch role {
		case model.RoleAdmin:
			if userRole != "admin" {
				return nil, gqlerrors.Forbidden("admin access required")
			}
		case model.RoleSeller:
			if userRole != "seller" && userRole != "admin" {
				return nil, gqlerrors.Forbidden("seller access required")
			}
		case model.RoleBuyer:
			if userRole == "" {
				return nil, gqlerrors.Forbidden("buyer access required")
			}
		}
		return next(ctx)
	}
}
