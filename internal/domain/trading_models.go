package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "NEW"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCanceled  OrderStatus = "CANCELED"
	OrderStatusPartially OrderStatus = "PARTIALLY_FILLED"
)

type Order struct {
	ID          uuid.UUID   `json:"id"`
	BybitID     string      `json:"bybit_id"`
	Symbol      string      `json:"symbol"`
	Side        OrderSide   `json:"side"`
	Type        OrderType   `json:"type"`
	Quantity    string      `json:"quantity"`
	Price       string      `json:"price,omitempty"`
	Status      OrderStatus `json:"status"`
	ExecutedQty string      `json:"executed_qty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	Symbol   string    `json:"symbol" binding:"required"`
	Side     OrderSide `json:"side" binding:"required"`
	Type     OrderType `json:"type" binding:"required"`
	Quantity string    `json:"quantity" binding:"required"`
	Price    string    `json:"price,omitempty"`
}

type TradeConfig struct {
	Symbol            string  `json:"symbol" binding:"required"`
	EntryVolume       string  `json:"entry_volume" binding:"required"`        // Объем входа
	DCAStepPercent    float64 `json:"dca_step_percent" binding:"required"`    // Шаг DCA в %
	DCAVolume         string  `json:"dca_volume" binding:"required"`          // Объем DCA ордеров
	DCACount          int     `json:"dca_count" binding:"required"`           // Количество DCA ордеров
	TakeProfitPercent float64 `json:"take_profit_percent" binding:"required"` // TP в %
	Martingale        float64 `json:"martingale"`                             // Мартингейл множитель
	DynamicStep       bool    `json:"dynamic_step"`                           // Динамический шаг цены
}

type Trade struct {
	ID              uuid.UUID   `json:"id"`
	Symbol          string      `json:"symbol"`
	Config          TradeConfig `json:"config"`
	EntryOrder      *Order      `json:"entry_order"`       // Ордер входа (market)
	DCAOrders       []Order     `json:"dca_orders"`        // Сетка DCA ордеров
	TakeProfitOrder *Order      `json:"take_profit_order"` // TP ордер
	Status          TradeStatus `json:"status"`
	TotalInvested   string      `json:"total_invested"`
	AveragePrice    string      `json:"average_price"`
	CurrentPrice    string      `json:"current_price"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type TradeStatus string

const (
	TradeStatusActive    TradeStatus = "ACTIVE"
	TradeStatusCompleted TradeStatus = "COMPLETED"
	TradeStatusCancelled TradeStatus = "CANCELLED"
	TradeStatusFailed    TradeStatus = "FAILED"
)
