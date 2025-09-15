package handler

import (
	"cryptorg/internal/domain"
	"cryptorg/internal/service"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type TradeHandler struct {
	tradeManager *service.TradeService
}

func (h *TradeHandler) bindJSON(ctx *fasthttp.RequestCtx, v interface{}) error {
	return json.Unmarshal(ctx.PostBody(), v)
}

func (h *TradeHandler) getParam(ctx *fasthttp.RequestCtx, key string) string {
	return ctx.UserValue(key).(string)
}

func (h *TradeHandler) sendResponse(ctx *fasthttp.RequestCtx, status int, data interface{}) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(status)

	if data != nil {
		json.NewEncoder(ctx).Encode(data)
	}
}

func (h *TradeHandler) sendError(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(status)
	ctx.Response.SetBodyString(`{"error": "` + message + `"}`)
}

func (h *TradeHandler) sendMessage(ctx *fasthttp.RequestCtx, message string) {
	h.sendResponse(ctx, 200, map[string]string{"message": message})
}

func NewTradeController(tradeManager *service.TradeService) *TradeHandler {
	return &TradeHandler{
		tradeManager: tradeManager,
	}
}

func (h *TradeHandler) InitializeTrade(ctx *fasthttp.RequestCtx) {
	var config domain.TradeConfig
	if err := h.bindJSON(ctx, &config); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	if config.Symbol == "" || config.EntryVolume == "" || config.DCAVolume == "" {
		h.sendError(ctx, 400, "Symbol, entry volume and DCA volume are required")
		return
	}

	if config.DCACount <= 0 || config.DCAStepPercent <= 0 || config.TakeProfitPercent <= 0 {
		h.sendError(ctx, 400, "DCA count, step percent and take profit percent must be positive")
		return
	}

	if config.Martingale <= 0 {
		config.Martingale = domain.DefaultMartingale
	}

	trade, err := h.tradeManager.InitializeTrade(ctx, config)
	if err != nil {
		h.sendError(ctx, 500, "Failed to initialize trade")
		return
	}

	h.sendResponse(ctx, 201, trade)
}

func (h *TradeHandler) GetTrade(ctx *fasthttp.RequestCtx) {
	tradeIDStr := h.getParam(ctx, "tradeId")
	if tradeIDStr == "" {
		h.sendError(ctx, 400, "Trade ID is required")
		return
	}

	tradeID, err := uuid.Parse(tradeIDStr)
	if err != nil {
		h.sendError(ctx, 400, "Invalid trade ID format")
		return
	}

	trade, err := h.tradeManager.GetTrade(tradeID)
	if err != nil {
		h.sendError(ctx, 404, "Trade not found")
		return
	}

	h.sendResponse(ctx, 200, trade)
}

func (h *TradeHandler) GetAllTrades(ctx *fasthttp.RequestCtx) {
	tradesMap := h.tradeManager.GetAllTrades()

	var trades []*domain.Trade
	for _, trade := range tradesMap {
		trades = append(trades, trade)
	}

	h.sendResponse(ctx, 200, map[string]interface{}{
		"trades": trades,
		"count":  len(trades),
	})
}

func (h *TradeHandler) ProcessOrderExecution(ctx *fasthttp.RequestCtx) {
	tradeIDStr := h.getParam(ctx, "tradeId")
	if tradeIDStr == "" {
		h.sendError(ctx, 400, "Trade ID is required")
		return
	}

	tradeID, err := uuid.Parse(tradeIDStr)
	if err != nil {
		h.sendError(ctx, 400, "Invalid trade ID format")
		return
	}

	var req struct {
		OrderID string `json:"order_id"`
	}

	if err := h.bindJSON(ctx, &req); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	if req.OrderID == "" {
		h.sendError(ctx, 400, "Order ID is required")
		return
	}

	if err := h.tradeManager.ProcessOrderExecution(ctx, tradeID, req.OrderID); err != nil {
		h.sendError(ctx, 500, "Failed to process order execution")
		return
	}

	h.sendMessage(ctx, "Order execution processed successfully")
}

func (h *TradeHandler) CloseTrade(ctx *fasthttp.RequestCtx) {
	tradeIDStr := h.getParam(ctx, "tradeId")
	if tradeIDStr == "" {
		h.sendError(ctx, 400, "Trade ID is required")
		return
	}

	tradeID, err := uuid.Parse(tradeIDStr)
	if err != nil {
		h.sendError(ctx, 400, "Invalid trade ID format")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := h.bindJSON(ctx, &req); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	reason := "Manual close"
	if req.Reason != "" {
		reason = req.Reason
	}

	if err := h.tradeManager.CloseTrade(ctx, tradeID, reason); err != nil {
		h.sendError(ctx, 500, "Failed to close trade")
		return
	}

	h.sendMessage(ctx, "Trade closed successfully")
}

func (h *TradeHandler) WebhookOrderUpdate(ctx *fasthttp.RequestCtx) {
	var webhookData struct {
		EventType string `json:"e"` // Event type
		Symbol    string `json:"s"` // Symbol
		OrderID   string `json:"i"` // Order ID
		Status    string `json:"X"` // Order status
		Side      string `json:"S"` // Side
		Type      string `json:"o"` // Order type
	}

	if err := h.bindJSON(ctx, &webhookData); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	if webhookData.Status == string(domain.OrderStatusBybitFilled) {
		trade, err := h.tradeManager.FindTradeByOrderID(webhookData.OrderID)
		if err != nil {
			h.sendMessage(ctx, "Order not found")
			return
		}

		orderType := h.determineOrderType(trade, webhookData.OrderID)

		if orderType == "entry" {
		} else {
			if err := h.tradeManager.ProcessOrderExecution(ctx, trade.ID, webhookData.OrderID); err != nil {
			}
		}
	}

	h.sendMessage(ctx, "Webhook processed")
}

func (h *TradeHandler) determineOrderType(trade *domain.Trade, orderID string) string {
	if trade.EntryOrder != nil && trade.EntryOrder.BybitID == orderID {
		return "entry"
	}

	if trade.TakeProfitOrder != nil && trade.TakeProfitOrder.BybitID == orderID {
		return "take_profit"
	}

	for _, dcaOrder := range trade.DCAOrders {
		if dcaOrder.BybitID == orderID {
			return "dca"
		}
	}

	return "unknown"
}
