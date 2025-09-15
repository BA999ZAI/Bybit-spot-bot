package router

import (
	"regexp"
	"strings"

	"cryptorg/internal/handler"

	"github.com/valyala/fasthttp"
)

type Router struct {
	orderController *handler.OrderHandler
	tradeController *handler.TradeHandler
	routes          []route
}

type route struct {
	method  string
	pattern *regexp.Regexp
	handler fasthttp.RequestHandler
	params  []string
}

func NewRouter(orderController *handler.OrderHandler, tradeController *handler.TradeHandler) *Router {
	r := &Router{
		orderController: orderController,
		tradeController: tradeController,
		routes:          make([]route, 0),
	}

	r.setupRoutes()
	return r
}

func (r *Router) Handler(ctx *fasthttp.RequestCtx) {
	r.setupCORS(ctx)

	if string(ctx.Method()) == "OPTIONS" {
		ctx.Response.SetStatusCode(204)
		return
	}

	method := string(ctx.Method())
	path := string(ctx.Path())

	for _, route := range r.routes {
		if route.method == method {
			matches := route.pattern.FindStringSubmatch(path)
			if matches != nil {
				for i, param := range route.params {
					if i+1 < len(matches) {
						ctx.SetUserValue(param, matches[i+1])
					}
				}
				route.handler(ctx)
				return
			}
		}
	}
	ctx.Response.SetStatusCode(404)
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetBodyString(`{"error": "Not Found", "message": "The requested resource was not found"}`)
}

func (r *Router) setupCORS(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func (r *Router) setupRoutes() {
	r.addRoute("GET", "/health", func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetStatusCode(200)
		ctx.Response.SetBodyString(`{"status": "ok", "service": "cryptorg-bot"}`)
	})

	r.addRoute("POST", "/api/orders/market", r.orderController.ExecuteMarketOrder)
	r.addRoute("POST", "/api/orders/limit", r.orderController.ExecuteLimitOrder)
	r.addRoute("DELETE", "/api/orders/([^/]+)/([^/]+)", r.orderController.TerminateOrder)
	r.addRoute("GET", "/api/orders/([^/]+)/([^/]+)", r.orderController.FetchOrderStatus)
	r.addRoute("POST", "/api/orders/calculate-tp", r.orderController.ComputeTakeProfit)
	r.addRoute("POST", "/api/orders/calculate-dca", r.orderController.ComputeDCAPrice)

	r.addRoute("POST", "/api/trades", r.tradeController.InitializeTrade)
	r.addRoute("GET", "/api/trades", r.tradeController.GetAllTrades)
	r.addRoute("POST", "/api/trades/([^/]+)/order-filled", r.tradeController.ProcessOrderExecution)
	r.addRoute("POST", "/api/trades/([^/]+)/close", r.tradeController.CloseTrade)
	r.addRoute("GET", "/api/trades/([^/]+)", r.tradeController.GetTrade)

	r.addRoute("POST", "/api/webhook/order-update", r.tradeController.WebhookOrderUpdate)
}

func (r *Router) addRoute(method, pattern string, handler fasthttp.RequestHandler) {
	regex, params := r.patternToRegex(pattern)
	r.routes = append(r.routes, route{
		method:  method,
		pattern: regex,
		handler: handler,
		params:  params,
	})
}

func (r *Router) patternToRegex(pattern string) (*regexp.Regexp, []string) {
	var params []string

	groupCount := strings.Count(pattern, "([^/]+)")

	if strings.Contains(pattern, "/api/orders/") && groupCount == 2 {
		params = []string{"symbol", "orderId"}
	} else if strings.Contains(pattern, "/api/trades/") && groupCount == 1 {
		params = []string{"tradeId"}
	} else if strings.Contains(pattern, "/api/trades/") && groupCount == 2 {
		params = []string{"tradeId", "action"}
	} else if groupCount > 0 {
		for i := 0; i < groupCount; i++ {
			params = append(params, "param"+string(rune('0'+i)))
		}
	}

	regex := regexp.MustCompile("^" + pattern + "$")
	return regex, params
}
