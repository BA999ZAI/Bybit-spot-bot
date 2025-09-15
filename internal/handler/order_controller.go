package handler

import (
	"cryptorg/internal/domain"
	"cryptorg/internal/service"
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type OrderHandler struct {
	orderManager *service.OrderService
}

func (h *OrderHandler) bindJSON(ctx *fasthttp.RequestCtx, v interface{}) error {
	return json.Unmarshal(ctx.PostBody(), v)
}

func (h *OrderHandler) getParam(ctx *fasthttp.RequestCtx, key string) string {
	return ctx.UserValue(key).(string)
}

func (h *OrderHandler) sendResponse(ctx *fasthttp.RequestCtx, status int, data interface{}) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(status)

	if data != nil {
		json.NewEncoder(ctx).Encode(data)
	}
}

func (h *OrderHandler) sendError(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(status)
	ctx.Response.SetBodyString(`{"error": "` + message + `"}`)
}

func (h *OrderHandler) sendMessage(ctx *fasthttp.RequestCtx, message string) {
	h.sendResponse(ctx, 200, map[string]string{"message": message})
}

func NewOrderController(orderManager *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderManager: orderManager,
	}
}

func (h *OrderHandler) ExecuteMarketOrder(ctx *fasthttp.RequestCtx) {
	var req domain.CreateOrderRequest
	if err := h.bindJSON(ctx, &req); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	req.Type = domain.OrderTypeMarket

	if req.Symbol == "" || req.Quantity == "" {
		h.sendError(ctx, 400, "Symbol and quantity are required")
		return
	}

	order, err := h.orderManager.ExecuteMarketOrder(ctx, req)
	if err != nil {
		h.sendError(ctx, 500, "Failed to execute market order")
		return
	}

	h.sendResponse(ctx, 201, order)
}

func (h *OrderHandler) ExecuteLimitOrder(ctx *fasthttp.RequestCtx) {
	var req domain.CreateOrderRequest
	if err := h.bindJSON(ctx, &req); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	req.Type = domain.OrderTypeLimit

	if req.Symbol == "" || req.Quantity == "" || req.Price == "" {
		h.sendError(ctx, 400, "Symbol, quantity and price are required")
		return
	}

	order, err := h.orderManager.ExecuteLimitOrder(ctx, req)
	if err != nil {
		h.sendError(ctx, 500, "Failed to execute limit order")
		return
	}

	h.sendResponse(ctx, 201, order)
}

func (h *OrderHandler) TerminateOrder(ctx *fasthttp.RequestCtx) {
	symbol := h.getParam(ctx, "symbol")
	orderIDStr := h.getParam(ctx, "orderId")

	if symbol == "" || orderIDStr == "" {
		h.sendError(ctx, 400, "Symbol and orderId are required")
		return
	}

	if err := h.orderManager.TerminateOrder(ctx, symbol, orderIDStr); err != nil {
		h.sendError(ctx, 500, "Failed to terminate order")
		return
	}

	h.sendMessage(ctx, "Order terminated successfully")
}

func (h *OrderHandler) FetchOrderStatus(ctx *fasthttp.RequestCtx) {
	symbol := h.getParam(ctx, "symbol")
	orderIDStr := h.getParam(ctx, "orderId")

	if symbol == "" || orderIDStr == "" {
		h.sendError(ctx, 400, "Symbol and orderId are required")
		return
	}

	order, err := h.orderManager.FetchOrderStatus(ctx, symbol, orderIDStr)
	if err != nil {
		h.sendError(ctx, 500, "Failed to fetch order status")
		return
	}

	h.sendResponse(ctx, 200, order)
}

func (h *OrderHandler) ComputeTakeProfit(ctx *fasthttp.RequestCtx) {
	var req struct {
		EntryPrice    string           `json:"entry_price"`
		ProfitPercent float64          `json:"profit_percent"`
		Side          domain.OrderSide `json:"side"`
	}

	if err := h.bindJSON(ctx, &req); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	if req.EntryPrice == "" || req.ProfitPercent <= 0 || req.Side == "" {
		h.sendError(ctx, 400, "All fields are required")
		return
	}

	tpPrice, err := h.orderManager.ComputeTakeProfitPrice(req.EntryPrice, req.ProfitPercent, req.Side)
	if err != nil {
		h.sendError(ctx, 500, "Failed to compute take profit price")
		return
	}

	h.sendResponse(ctx, 200, map[string]interface{}{
		"entry_price":       req.EntryPrice,
		"profit_percent":    req.ProfitPercent,
		"side":              req.Side,
		"take_profit_price": tpPrice,
	})
}

func (h *OrderHandler) ComputeDCAPrice(ctx *fasthttp.RequestCtx) {
	var req struct {
		CurrentPrice string           `json:"current_price"`
		StepPercent  float64          `json:"step_percent"`
		Side         domain.OrderSide `json:"side"`
	}

	if err := h.bindJSON(ctx, &req); err != nil {
		h.sendError(ctx, 400, "Invalid JSON")
		return
	}

	if req.CurrentPrice == "" || req.StepPercent <= 0 || req.Side == "" {
		h.sendError(ctx, 400, "All fields are required")
		return
	}

	dcaPrice, err := h.orderManager.ComputeDCAPrice(req.CurrentPrice, req.StepPercent, req.Side)
	if err != nil {
		h.sendError(ctx, 500, "Failed to compute DCA price")
		return
	}

	h.sendResponse(ctx, 200, map[string]interface{}{
		"current_price": req.CurrentPrice,
		"step_percent":  req.StepPercent,
		"side":          req.Side,
		"dca_price":     dcaPrice,
	})
}
