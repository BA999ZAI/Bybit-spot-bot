package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"cryptorg/internal/domain"

	"github.com/google/uuid"
)

type TradeService struct {
	orderManager *OrderService
	trades       map[uuid.UUID]*domain.Trade
	orderIndex   map[string]uuid.UUID // orderID -> tradeID для быстрого поиска
	mu           sync.RWMutex
}

func NewTradeManager(orderManager *OrderService) *TradeService {
	return &TradeService{
		orderManager: orderManager,
		trades:       make(map[uuid.UUID]*domain.Trade),
		orderIndex:   make(map[string]uuid.UUID),
	}
}

func (s *TradeService) InitializeTrade(ctx context.Context, config domain.TradeConfig) (*domain.Trade, error) {
	entryOrderReq := domain.CreateOrderRequest{
		Symbol:   config.Symbol,
		Side:     domain.OrderSideBuy,
		Type:     domain.OrderTypeMarket,
		Quantity: config.EntryVolume,
	}

	entryOrder, err := s.orderManager.ExecuteMarketOrder(ctx, entryOrderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute entry order: %w", err)
	}

	trade := &domain.Trade{
		ID:            uuid.New(),
		Symbol:        config.Symbol,
		Config:        config,
		EntryOrder:    entryOrder,
		DCAOrders:     make([]domain.Order, 0),
		Status:        domain.TradeStatusActive,
		TotalInvested: config.EntryVolume,
		AveragePrice:  entryOrder.Price,
		CurrentPrice:  entryOrder.Price,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.setupTakeProfitOrder(ctx, trade); err != nil {
	}

	if err := s.setupDCAOrders(ctx, trade); err != nil {
	}

	s.mu.Lock()
	s.trades[trade.ID] = trade
	s.indexOrders(trade)
	s.mu.Unlock()

	return trade, nil
}

func (s *TradeService) setupTakeProfitOrder(ctx context.Context, trade *domain.Trade) error {
	entryPrice, err := strconv.ParseFloat(trade.EntryOrder.Price, 64)
	if err != nil {
		return fmt.Errorf("invalid entry price: %w", err)
	}

	tpPrice := entryPrice * (1 + trade.Config.TakeProfitPercent/100)
	tpPriceStr := fmt.Sprintf("%.8f", tpPrice)

	totalVolume := trade.Config.EntryVolume
	if trade.Config.Martingale > 0 {
		for i := 0; i < trade.Config.DCACount; i++ {
			dcaVolume, _ := strconv.ParseFloat(trade.Config.DCAVolume, 64)
			totalVolumeFloat, _ := strconv.ParseFloat(totalVolume, 64)
			totalVolumeFloat += dcaVolume * trade.Config.Martingale
			totalVolume = fmt.Sprintf("%.8f", totalVolumeFloat)
		}
	}

	tpOrderReq := domain.CreateOrderRequest{
		Symbol:   trade.Config.Symbol,
		Side:     domain.OrderSideSell,
		Type:     domain.OrderTypeLimit,
		Quantity: totalVolume,
		Price:    tpPriceStr,
	}

	tpOrder, err := s.orderManager.ExecuteLimitOrder(ctx, tpOrderReq)
	if err != nil {
		return fmt.Errorf("failed to create take profit order: %w", err)
	}

	trade.TakeProfitOrder = tpOrder
	return nil
}

func (s *TradeService) setupDCAOrders(ctx context.Context, trade *domain.Trade) error {
	entryPrice, err := strconv.ParseFloat(trade.EntryOrder.Price, 64)
	if err != nil {
		return fmt.Errorf("invalid entry price: %w", err)
	}

	currentPrice := entryPrice
	currentVolume := trade.Config.DCAVolume

	for i := 0; i < trade.Config.DCACount; i++ {
		if trade.Config.DynamicStep {
			stepPercent := trade.Config.DCAStepPercent * float64(i+1)
			dcaPrice := currentPrice * (1 - stepPercent/100)
			currentPrice = dcaPrice
		} else {
			dcaPrice := currentPrice * (1 - trade.Config.DCAStepPercent/100)
			currentPrice = dcaPrice
		}

		dcaPriceStr := fmt.Sprintf("%.8f", currentPrice)

		if trade.Config.Martingale > 0 {
			volumeFloat, _ := strconv.ParseFloat(currentVolume, 64)
			volumeFloat *= trade.Config.Martingale
			currentVolume = fmt.Sprintf("%.8f", volumeFloat)
		}

		dcaOrderReq := domain.CreateOrderRequest{
			Symbol:   trade.Config.Symbol,
			Side:     domain.OrderSideBuy,
			Type:     domain.OrderTypeLimit,
			Quantity: currentVolume,
			Price:    dcaPriceStr,
		}

		dcaOrder, err := s.orderManager.ExecuteLimitOrder(ctx, dcaOrderReq)
		if err != nil {
			continue
		}

		trade.DCAOrders = append(trade.DCAOrders, *dcaOrder)
	}

	return nil
}

func (s *TradeService) indexOrders(trade *domain.Trade) {
	if trade.EntryOrder != nil {
		s.orderIndex[trade.EntryOrder.BybitID] = trade.ID
	}

	if trade.TakeProfitOrder != nil {
		s.orderIndex[trade.TakeProfitOrder.BybitID] = trade.ID
	}

	for _, dcaOrder := range trade.DCAOrders {
		s.orderIndex[dcaOrder.BybitID] = trade.ID
	}
}

func (s *TradeService) unindexOrders(trade *domain.Trade) {
	if trade.EntryOrder != nil {
		delete(s.orderIndex, trade.EntryOrder.BybitID)
	}

	if trade.TakeProfitOrder != nil {
		delete(s.orderIndex, trade.TakeProfitOrder.BybitID)
	}

	for _, dcaOrder := range trade.DCAOrders {
		delete(s.orderIndex, dcaOrder.BybitID)
	}
}

func (s *TradeService) FindTradeByOrderID(orderID string) (*domain.Trade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tradeID, exists := s.orderIndex[orderID]
	if !exists {
		return nil, fmt.Errorf("trade not found for order ID: %s", orderID)
	}

	trade, exists := s.trades[tradeID]
	if !exists {
		return nil, fmt.Errorf("trade %s not found", tradeID)
	}

	return trade, nil
}

func (s *TradeService) ProcessOrderExecution(ctx context.Context, tradeID uuid.UUID, orderID string) error {
	s.mu.Lock()
	trade, exists := s.trades[tradeID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("trade not found: %s", tradeID)
	}

	if trade.TakeProfitOrder != nil && trade.TakeProfitOrder.BybitID == orderID {
		return s.finalizeTrade(ctx, tradeID, domain.TradeStatusCompleted)
	}

	for i, dcaOrder := range trade.DCAOrders {
		if dcaOrder.BybitID == orderID {
			return s.handleDCAExecution(ctx, trade, i)
		}
	}

	return fmt.Errorf("order %s not found in trade %s", orderID, tradeID)
}

func (s *TradeService) handleDCAExecution(ctx context.Context, trade *domain.Trade, dcaOrderIndex int) error {
	dcaOrder := &trade.DCAOrders[dcaOrderIndex]

	updatedOrder, err := s.orderManager.FetchOrderStatus(ctx, dcaOrder.Symbol, dcaOrder.BybitID)
	if err != nil {
		return fmt.Errorf("failed to get updated DCA order status: %w", err)
	}

	trade.DCAOrders[dcaOrderIndex] = *updatedOrder

	if err := s.updateTakeProfitOrder(ctx, trade); err != nil {
	}

	trade.UpdatedAt = time.Now()
	return nil
}

func (s *TradeService) updateTakeProfitOrder(ctx context.Context, trade *domain.Trade) error {
	if trade.TakeProfitOrder != nil {
		if err := s.orderManager.TerminateOrder(ctx, trade.Symbol, trade.TakeProfitOrder.BybitID); err != nil {
		}
	}

	newAveragePrice, totalVolume, err := s.calculateNewAveragePrice(trade)
	if err != nil {
		return fmt.Errorf("failed to calculate new average price: %w", err)
	}

	tpPrice := newAveragePrice * (1 + trade.Config.TakeProfitPercent/100)
	tpPriceStr := fmt.Sprintf("%.8f", tpPrice)

	tpOrderReq := domain.CreateOrderRequest{
		Symbol:   trade.Config.Symbol,
		Side:     domain.OrderSideSell,
		Type:     domain.OrderTypeLimit,
		Quantity: totalVolume,
		Price:    tpPriceStr,
	}

	tpOrder, err := s.orderManager.ExecuteLimitOrder(ctx, tpOrderReq)
	if err != nil {
		return fmt.Errorf("failed to create new take profit order: %w", err)
	}

	trade.TakeProfitOrder = tpOrder
	trade.AveragePrice = fmt.Sprintf("%.8f", newAveragePrice)
	return nil
}

func (s *TradeService) calculateNewAveragePrice(trade *domain.Trade) (float64, string, error) {
	totalVolume := 0.0
	totalCost := 0.0

	entryVolume, err := strconv.ParseFloat(trade.EntryOrder.Quantity, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid entry volume: %w", err)
	}
	entryPrice, err := strconv.ParseFloat(trade.EntryOrder.Price, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid entry price: %w", err)
	}

	totalVolume += entryVolume
	totalCost += entryVolume * entryPrice

	for _, dcaOrder := range trade.DCAOrders {
		if dcaOrder.Status == domain.OrderStatusFilled {
			dcaVolume, err := strconv.ParseFloat(dcaOrder.ExecutedQty, 64)
			if err != nil {
				continue
			}
			dcaPrice, err := strconv.ParseFloat(dcaOrder.Price, 64)
			if err != nil {
				continue
			}

			totalVolume += dcaVolume
			totalCost += dcaVolume * dcaPrice
		}
	}

	if totalVolume == 0 {
		return 0, "", fmt.Errorf("total volume is zero")
	}

	averagePrice := totalCost / totalVolume
	totalVolumeStr := fmt.Sprintf("%.8f", totalVolume)

	return averagePrice, totalVolumeStr, nil
}

func (s *TradeService) finalizeTrade(ctx context.Context, tradeID uuid.UUID, status domain.TradeStatus) error {
	s.mu.Lock()
	trade, exists := s.trades[tradeID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("trade not found: %s", tradeID)
	}

	trade.Status = status
	trade.UpdatedAt = time.Now()

	s.unindexOrders(trade)
	s.mu.Unlock()

	for _, dcaOrder := range trade.DCAOrders {
		if dcaOrder.Status == domain.OrderStatusNew {
			if err := s.orderManager.TerminateOrder(ctx, dcaOrder.Symbol, dcaOrder.BybitID); err != nil {
			}
		}
	}

	return nil
}

func (s *TradeService) GetTrade(tradeID uuid.UUID) (*domain.Trade, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trade, exists := s.trades[tradeID]
	if !exists {
		return nil, fmt.Errorf("trade not found: %s", tradeID)
	}

	return trade, nil
}

func (s *TradeService) GetAllTrades() map[uuid.UUID]*domain.Trade {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[uuid.UUID]*domain.Trade)
	for id, trade := range s.trades {
		result[id] = trade
	}

	return result
}

func (s *TradeService) CloseTrade(ctx context.Context, tradeID uuid.UUID, reason string) error {
	s.mu.RLock()
	_, exists := s.trades[tradeID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("trade not found: %s", tradeID)
	}

	return s.finalizeTrade(ctx, tradeID, domain.TradeStatusCancelled)
}
