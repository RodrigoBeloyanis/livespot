package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
)

type BinanceOrderClient struct {
	client *binance.Client
}

func NewBinanceOrderClient(client *binance.Client) *BinanceOrderClient {
	return &BinanceOrderClient{client: client}
}

func (b *BinanceOrderClient) SubmitOrder(ctx context.Context, req OrderRequest) (OrderResponse, error) {
	params, err := buildOrderParams(req)
	if err != nil {
		return OrderResponse{}, err
	}
	resp, err := b.client.NewOrder(ctx, params)
	if err != nil {
		return OrderResponse{}, err
	}
	return parseOrderResponse(resp.Body)
}

func (b *BinanceOrderClient) CancelOrder(ctx context.Context, req CancelRequest) (OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("origClientOrderId", req.ClientOrderID)
	resp, err := b.client.CancelOrder(ctx, params)
	if err != nil {
		return OrderResponse{}, err
	}
	return parseOrderResponse(resp.Body)
}

func (b *BinanceOrderClient) CancelReplaceOrder(ctx context.Context, req CancelReplaceRequest) (OrderResponse, error) {
	if req.Side == "" || req.Type == "" || req.TimeInForce == "" {
		return OrderResponse{}, fmt.Errorf("cancel replace missing fields")
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("cancelOrigClientOrderId", req.ClientOrderID)
	params.Set("newClientOrderId", req.NewClientID)
	params.Set("newOrderRespType", "RESULT")
	params.Set("cancelReplaceMode", "STOP_ON_FAILURE")
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	params.Set("timeInForce", string(req.TimeInForce))
	params.Set("price", req.NewPrice)
	params.Set("quantity", req.NewQty)
	resp, err := b.client.CancelReplaceOrder(ctx, params)
	if err != nil {
		return OrderResponse{}, err
	}
	return parseOrderResponse(resp.Body)
}

func (b *BinanceOrderClient) SubmitOCO(ctx context.Context, req OCORequest) (OCOResponse, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", string(req.Side))
	params.Set("quantity", req.Qty)
	params.Set("price", req.TPPrice)
	params.Set("stopPrice", req.SLStopPrice)
	params.Set("stopLimitPrice", req.SLStopLimitPrice)
	params.Set("stopLimitTimeInForce", string(req.SLStopLimitTIF))
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", req.LimitClientOrderID)
	}
	if req.StopClientOrderID != "" {
		params.Set("stopClientOrderId", req.StopClientOrderID)
	}
	resp, err := b.client.NewOCO(ctx, params)
	if err != nil {
		return OCOResponse{}, err
	}
	return parseOCOResponse(resp.Body)
}

func (b *BinanceOrderClient) GetOrderByClientID(ctx context.Context, symbol string, clientOrderID string) (OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("origClientOrderId", clientOrderID)
	resp, err := b.client.QueryOrder(ctx, params)
	if err != nil {
		if be, ok := err.(binance.BinanceError); ok && be.Code == -2013 {
			return OrderResponse{Found: false}, nil
		}
		return OrderResponse{}, err
	}
	order, err := parseOrderResponse(resp.Body)
	if err != nil {
		return OrderResponse{}, err
	}
	order.Found = true
	return order, nil
}

type binanceOrderAck struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	Status        string `json:"status"`
}

type binanceOCOAck struct {
	OrderListID      int64  `json:"orderListId"`
	ListClientOrderID string `json:"listClientOrderId"`
	ListOrderStatus  string `json:"listOrderStatus"`
}

func parseOrderResponse(body []byte) (OrderResponse, error) {
	var ack binanceOrderAck
	if err := json.Unmarshal(body, &ack); err != nil {
		return OrderResponse{}, fmt.Errorf("order decode: %w", err)
	}
	resp := OrderResponse{
		Found:         true,
		Rejected:      strings.EqualFold(ack.Status, "REJECTED"),
		OrderID:       fmt.Sprintf("%d", ack.OrderID),
		ClientOrderID: ack.ClientOrderID,
		Status:        ack.Status,
	}
	return resp, nil
}

func parseOCOResponse(body []byte) (OCOResponse, error) {
	var ack binanceOCOAck
	if err := json.Unmarshal(body, &ack); err != nil {
		return OCOResponse{}, fmt.Errorf("oco decode: %w", err)
	}
	resp := OCOResponse{
		Rejected:     strings.EqualFold(ack.ListOrderStatus, "REJECTED"),
		OrderListID:  fmt.Sprintf("%d", ack.OrderListID),
		ListClientID: ack.ListClientOrderID,
		Status:       ack.ListOrderStatus,
	}
	return resp, nil
}

func buildOrderParams(req OrderRequest) (url.Values, error) {
	if req.Symbol == "" || req.Side == "" || req.Type == "" {
		return nil, fmt.Errorf("order params missing")
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	if req.ClientOrderID != "" {
		params.Set("newClientOrderId", req.ClientOrderID)
	}
	if req.Type == OrderTypeLimit || req.Type == OrderTypeLimitMaker {
		if req.Price == "" || req.Qty == "" {
			return nil, fmt.Errorf("limit price/qty missing")
		}
		params.Set("price", req.Price)
		params.Set("quantity", req.Qty)
		if req.Type == OrderTypeLimit {
			params.Set("timeInForce", string(req.TimeInForce))
		}
		return params, nil
	}
	if req.Type == OrderTypeMarket {
		if req.Qty == "" {
			return nil, fmt.Errorf("market qty missing")
		}
		params.Set("quantity", req.Qty)
		return params, nil
	}
	return nil, fmt.Errorf("order type unsupported")
}
