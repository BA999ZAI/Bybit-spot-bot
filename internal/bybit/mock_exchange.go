package bybit

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) ExecuteOrder(ctx context.Context, req ExchangeOrderRequest) (*ExchangeOrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExchangeOrderResponse), args.Error(1)
}

func (m *MockClient) TerminateOrder(ctx context.Context, req ExchangeCancelRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockClient) FetchOrderInfo(ctx context.Context, symbol string, orderID string) (*ExchangeOrderResponse, error) {
	args := m.Called(ctx, symbol, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExchangeOrderResponse), args.Error(1)
}
