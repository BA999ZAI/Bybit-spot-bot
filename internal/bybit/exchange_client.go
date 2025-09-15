package bybit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	apiKey     string
	secretKey  string
	testnet    bool
	httpClient *http.Client
}

func NewExchangeClient(apiKey, secretKey string, testnet bool) *Client {
	return &Client{
		apiKey:     apiKey,
		secretKey:  secretKey,
		testnet:    testnet,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) getBaseURL() string {
	if c.testnet {
		return "https://api-testnet.bybit.com"
	}
	return "https://api.bybit.com"
}

type ExchangeOrderRequest struct {
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	OrderType   string `json:"orderType"`
	Qty         string `json:"qty,omitempty"`
	Price       string `json:"price,omitempty"`
	TimeInForce string `json:"timeInForce,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

type ExchangeOrderResponse struct {
	Symbol      string `json:"symbol"`
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
	Price       string `json:"price"`
	Qty         string `json:"qty"`
	ExecutedQty string `json:"executedQty"`
	Status      string `json:"orderStatus"`
	TimeInForce string `json:"timeInForce"`
	OrderType   string `json:"orderType"`
	Side        string `json:"side"`
	CreatedTime string `json:"createdTime"`
}

type ExchangeCancelRequest struct {
	Symbol    string `json:"symbol"`
	OrderID   string `json:"orderId,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func (c *Client) ExecuteOrder(ctx context.Context, req ExchangeOrderRequest) (*ExchangeOrderResponse, error) {
	req.Timestamp = time.Now().UnixMilli()

	endpoint := "/v5/order/create"

	resp, err := c.makeAuthenticatedRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bybit API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Result ExchangeOrderResponse `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}
	return &apiResp.Result, nil
}

func (c *Client) TerminateOrder(ctx context.Context, req ExchangeCancelRequest) error {
	req.Timestamp = time.Now().UnixMilli()

	endpoint := "/v5/order/cancel"

	resp, err := c.makeAuthenticatedRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bybit API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) FetchOrderInfo(ctx context.Context, symbol string, orderID string) (*ExchangeOrderResponse, error) {
	timestamp := time.Now().UnixMilli()

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderID)
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))

	endpoint := "/v5/order/realtime?" + params.Encode()
	signature := c.createSignature(params.Encode())
	endpoint += "&signature=" + signature

	req, err := http.NewRequestWithContext(ctx, "GET", c.getBaseURL()+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-BAPI-API-KEY", c.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(timestamp, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bybit API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Result struct {
			List []ExchangeOrderResponse `json:"list"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	if len(apiResp.Result.List) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	return &apiResp.Result.List[0], nil
}

func (c *Client) makeAuthenticatedRequest(ctx context.Context, method, endpoint string, payload interface{}) (*http.Response, error) {
	timestamp := time.Now().UnixMilli()

	var body io.Reader
	var queryString string

	if method == "GET" || method == "DELETE" {
		params, err := structToURLValues(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to convert payload to query params: %w", err)
		}
		queryString = params.Encode()
	} else {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
		queryString = string(jsonData)
	}

	signature := c.createSignature(strconv.FormatInt(timestamp, 10) + c.apiKey + queryString)

	var requestURL string
	if method == "GET" || method == "DELETE" {
		requestURL = c.getBaseURL() + endpoint + "?" + queryString
	} else {
		requestURL = c.getBaseURL() + endpoint
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-BAPI-API-KEY", c.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-BAPI-RECV-WINDOW", "5000")

	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func (c *Client) createSignature(queryString string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

func structToURLValues(v interface{}) (url.Values, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	values := url.Values{}
	for key, val := range m {
		if val != nil {
			switch v := val.(type) {
			case string:
				if v != "" {
					values.Set(key, v)
				}
			case float64:
				values.Set(key, strconv.FormatFloat(v, 'f', -1, 64))
			case bool:
				values.Set(key, strconv.FormatBool(v))
			default:
				values.Set(key, fmt.Sprintf("%v", v))
			}
		}
	}

	return values, nil
}
