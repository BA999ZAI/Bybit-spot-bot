package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cryptorg/internal/bybit"
	"cryptorg/internal/handler"
	"cryptorg/internal/router"
	"cryptorg/internal/service"
	"cryptorg/pkg/config"

	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

type App struct {
	config          *config.Config
	exchangeClient  *bybit.Client
	orderManager    *service.OrderService
	tradeManager    *service.TradeService
	orderController *handler.OrderHandler
	tradeController *handler.TradeHandler
	router          *router.Router
	server          *fasthttp.Server
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}
}

func NewApplication() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	exchangeClient := bybit.NewExchangeClient(
		cfg.Bybit.APIKey,
		cfg.Bybit.SecretKey,
		cfg.Bybit.Testnet,
	)

	orderManager := service.NewOrderManager(exchangeClient)
	tradeManager := service.NewTradeManager(orderManager)

	orderController := handler.NewOrderController(orderManager)
	tradeController := handler.NewTradeController(tradeManager)

	appRouter := router.NewRouter(orderController, tradeController)

	server := &fasthttp.Server{
		Handler:      appRouter.Handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	app := &App{
		config:          cfg,
		exchangeClient:  exchangeClient,
		orderManager:    orderManager,
		tradeManager:    tradeManager,
		orderController: orderController,
		tradeController: tradeController,
		router:          appRouter,
		server:          server,
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("Starting Cryptorg Bot on port %s", a.config.Server.Port)
	log.Printf("Environment: %s", a.config.Base.Environment)
	log.Printf("Bybit Testnet: %v", a.config.Bybit.Testnet)
	log.Printf("Symbol: %s", a.config.Bybit.Symbol)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	addr := ":" + a.config.Server.Port
	go func() {
		log.Printf("FastHTTP server starting on %s", addr)
		if err := a.server.ListenAndServe(addr); err != nil {
			log.Fatalf("Failed to start FastHTTP server: %v", err)
		}
	}()

	log.Println("Cryptorg Bot started successfully")

	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down...")
	case sig := <-quit:
		log.Printf("Received shutdown signal: %v", sig)
	}

	return a.shutdown()
}

func (a *App) shutdown() error {
	log.Println("Shutting down Cryptorg Bot...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.ShutdownWithContext(ctx); err != nil {
		log.Printf("Failed to shutdown FastHTTP server gracefully: %v", err)
		return err
	}

	log.Println("Cryptorg Bot shut down successfully")
	return nil
}

func (a *App) GetConfig() *config.Config {
	return a.config
}

func (a *App) GetOrderManager() *service.OrderService {
	return a.orderManager
}

func (a *App) GetTradeManager() *service.TradeService {
	return a.tradeManager
}
