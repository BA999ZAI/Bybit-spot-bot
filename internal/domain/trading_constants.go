package domain
type OrderStatusBybit string

const (
	OrderStatusBybitFilled   OrderStatusBybit = "Filled"
	OrderStatusBybitNew      OrderStatusBybit = "New"
	OrderStatusBybitCanceled OrderStatusBybit = "Cancelled"
)

const (
	DefaultMartingale  = 1.0
	DefaultTimeInForce = "GTC"
	PricePrecision     = 8
	MaxSafetyOrders    = 20
	MaxPositionValue   = 100000.0
	MinOrderSize       = 0.001
	MaxOrderSize       = 1000.0
)

const (
	SymbolPattern = `^[A-Z]{3,10}USDT$`
	MinPriceStep  = 0.1
	MaxPriceStep  = 50.0
	MinProfitStep = 0.1
	MaxProfitStep = 100.0
)

const (
	WebhookEventOrderUpdate = "executionReport"
)

const (
	HTTPStatusOK                  = 200
	HTTPStatusCreated             = 201
	HTTPStatusNoContent           = 204
	HTTPStatusBadRequest          = 400
	HTTPStatusNotFound            = 404
	HTTPStatusUnprocessableEntity = 422
	HTTPStatusInternalServerError = 500
	HTTPStatusBadGateway          = 502
)
