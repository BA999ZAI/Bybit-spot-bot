package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cryptorg/internal/bybit"
	"cryptorg/internal/domain"

	"github.com/google/uuid"
)

type OrderService struct {
	exchangeClient *bybit.Client
}

func NewOrderManager(exchangeClient *bybit.Client) *OrderService {
	return &OrderService{
		exchangeClient: exchangeClient,
	}
}

func (s *OrderService) ExecuteMarketOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	exchangeReq := bybit.ExchangeOrderRequest{
		Symbol:    req.Symbol,
		Side:      string(req.Side),
		OrderType: string(req.Type),
		Qty:       req.Quantity,
		Timestamp: time.Now().UnixMilli(),
	}

	exchangeResp, err := s.exchangeClient.ExecuteOrder(ctx, exchangeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute market order: %w", err)
	}

	order := s.buildOrderFromResponse(exchangeResp)
	return order, nil
}

func (s *OrderService) ExecuteLimitOrder(ctx context.Context, req domain.CreateOrderRequest) (*domain.Order, error) {
	if req.Price == "" {
		return nil, fmt.Errorf("price is required for limit order")
	}

	quantity, err := s.calculateQuantityFromUSDT(req.Quantity, req.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate quantity: %w", err)
	}

	exchangeReq := bybit.ExchangeOrderRequest{
		Symbol:      req.Symbol,
		Side:        string(req.Side),
		OrderType:   string(req.Type),
		Qty:         quantity,
		Price:       req.Price,
		TimeInForce: domain.DefaultTimeInForce,
		Timestamp:   time.Now().UnixMilli(),
	}

	exchangeResp, err := s.exchangeClient.ExecuteOrder(ctx, exchangeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute limit order: %w", err)
	}

	order := s.buildOrderFromResponse(exchangeResp)
	return order, nil
}

func (s *OrderService) TerminateOrder(ctx context.Context, symbol string, orderID string) error {
	cancelReq := bybit.ExchangeCancelRequest{
		Symbol:    symbol,
		OrderID:   orderID,
		Timestamp: time.Now().UnixMilli(),
	}

	if err := s.exchangeClient.TerminateOrder(ctx, cancelReq); err != nil {
		return fmt.Errorf("failed to terminate order: %w", err)
	}

	return nil
}

func (s *OrderService) FetchOrderStatus(ctx context.Context, symbol string, orderID string) (*domain.Order, error) {
	exchangeResp, err := s.exchangeClient.FetchOrderInfo(ctx, symbol, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status: %w", err)
	}

	order := s.buildOrderFromResponse(exchangeResp)
	return order, nil
}

func (s *OrderService) ComputeTakeProfitPrice(entryPrice string, profitPercent float64, side domain.OrderSide) (string, error) {
	if entryPrice == "" {
		return "", fmt.Errorf("entry price is required")
	}
	if profitPercent <= 0 {
		return "", fmt.Errorf("profit percent must be positive")
	}

	price, err := strconv.ParseFloat(entryPrice, 64)
	if err != nil {
		return "", fmt.Errorf("invalid entry price: %w", err)
	}

	var tpPrice float64
	if side == domain.OrderSideBuy {
		tpPrice = price * (1 + profitPercent/100)
	} else {
		tpPrice = price * (1 - profitPercent/100)
	}

	return fmt.Sprintf("%.8f", tpPrice), nil
}

func (s *OrderService) ComputeDCAPrice(currentPrice string, stepPercent float64, side domain.OrderSide) (string, error) {
	if currentPrice == "" {
		return "", fmt.Errorf("current price is required")
	}
	if stepPercent <= 0 {
		return "", fmt.Errorf("step percent must be positive")
	}

	price, err := strconv.ParseFloat(currentPrice, 64)
	if err != nil {
		return "", fmt.Errorf("invalid current price: %w", err)
	}

	var dcaPrice float64
	if side == domain.OrderSideBuy {
		dcaPrice = price * (1 - stepPercent/100)
	} else {
		dcaPrice = price * (1 + stepPercent/100)
	}

	return fmt.Sprintf("%.8f", dcaPrice), nil
}

func (s *OrderService) calculateQuantityFromUSDT(usdtAmount, price string) (string, error) {
	usdt, err := strconv.ParseFloat(usdtAmount, 64)
	if err != nil {
		return "", fmt.Errorf("invalid USDT amount: %w", err)
	}

	priceFloat, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "", fmt.Errorf("invalid price: %w", err)
	}

	if priceFloat <= 0 {
		return "", fmt.Errorf("price must be positive")
	}

	quantity := usdt / priceFloat
	return fmt.Sprintf("%.8f", quantity), nil
}

func (s *OrderService) buildOrderFromResponse(resp *bybit.ExchangeOrderResponse) *domain.Order {
	return &domain.Order{
		ID:          uuid.New(),
		BybitID:     resp.OrderID,
		Symbol:      resp.Symbol,
		Side:        domain.OrderSide(resp.Side),
		Type:        domain.OrderType(resp.OrderType),
		Quantity:    resp.Qty,
		Price:       resp.Price,
		Status:      domain.OrderStatus(resp.Status),
		ExecutedQty: resp.ExecutedQty,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
