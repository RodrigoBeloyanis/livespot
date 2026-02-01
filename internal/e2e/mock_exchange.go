package e2e

import (
	"errors"
	"time"
)

type MockExchange struct {
	Now           func() time.Time
	FiltersJSON   []byte
	DepthBySymbol map[string][]byte
	BalancesJSON  []byte
	orders        map[string]OrderResponse
}

func NewMockExchange(now func() time.Time) *MockExchange {
	if now == nil {
		now = time.Now
	}
	return &MockExchange{
		Now:           now,
		DepthBySymbol: map[string][]byte{},
		orders:        map[string]OrderResponse{},
	}
}

func (m *MockExchange) Time(ctx Context) (int64, error) {
	return m.Now().UnixMilli(), nil
}

func (m *MockExchange) ExchangeInfo(ctx Context) ([]byte, error) {
	return m.FiltersJSON, nil
}

func (m *MockExchange) Depth(ctx Context, symbol string, limit int) ([]byte, error) {
	if payload, ok := m.DepthBySymbol[symbol]; ok {
		return payload, nil
	}
	return nil, errors.New("depth not found")
}

func (m *MockExchange) NewOrder(ctx Context, req OrderRequest) (OrderResponse, error) {
	now := m.Now().UnixMilli()
	resp := OrderResponse{
		Symbol:        req.Symbol,
		ClientOrderID: req.ClientOrderID,
		Status:        "NEW",
		Price:         req.Price,
		Qty:           req.Qty,
		Side:          req.Side,
		CreatedTsMs:   now,
		UpdatedTsMs:   now,
	}
	m.orders[req.ClientOrderID] = resp
	return resp, nil
}

func (m *MockExchange) QueryOrder(ctx Context, symbol string, clientOrderID string) (OrderResponse, error) {
	resp, ok := m.orders[clientOrderID]
	if !ok {
		return OrderResponse{}, errors.New("order not found")
	}
	return resp, nil
}

func (m *MockExchange) CancelOrder(ctx Context, symbol string, clientOrderID string) (OrderResponse, error) {
	resp, ok := m.orders[clientOrderID]
	if !ok {
		return OrderResponse{}, errors.New("order not found")
	}
	resp.Status = "CANCELED"
	resp.UpdatedTsMs = m.Now().UnixMilli()
	m.orders[clientOrderID] = resp
	return resp, nil
}

func (m *MockExchange) CancelReplace(ctx Context, req CancelReplaceRequest) (OrderResponse, error) {
	if req.CancelClientOrderID != "" {
		if _, err := m.CancelOrder(ctx, req.Symbol, req.CancelClientOrderID); err != nil {
			return OrderResponse{}, err
		}
	}
	newID := req.NewClientOrderID
	if newID == "" {
		newID = req.NewClientOrderIDHint
	}
	if newID == "" {
		return OrderResponse{}, errors.New("new client order id missing")
	}
	return m.NewOrder(ctx, OrderRequest{
		Symbol:        req.Symbol,
		Side:          req.Side,
		Price:         req.Price,
		Qty:           req.Qty,
		ClientOrderID: newID,
	})
}

func (m *MockExchange) OpenOrders(ctx Context, symbol string) ([]OrderResponse, error) {
	out := make([]OrderResponse, 0, len(m.orders))
	for _, order := range m.orders {
		if order.Symbol == symbol && order.Status == "NEW" {
			out = append(out, order)
		}
	}
	return out, nil
}

func (m *MockExchange) AllOrders(ctx Context, symbol string, limit int) ([]OrderResponse, error) {
	out := make([]OrderResponse, 0, len(m.orders))
	for _, order := range m.orders {
		if symbol == "" || order.Symbol == symbol {
			out = append(out, order)
		}
	}
	return out, nil
}

func (m *MockExchange) Account(ctx Context) ([]byte, error) {
	return m.BalancesJSON, nil
}
