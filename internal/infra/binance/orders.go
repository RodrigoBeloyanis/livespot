package binance

import "encoding/json"

type OpenOrder struct {
	Symbol        string `json:"symbol"`
	ClientOrderID string `json:"clientOrderId"`
	Status        string `json:"status"`
	OrderID       int64  `json:"orderId"`
}

func ParseOpenOrders(body []byte) ([]OpenOrder, error) {
	var orders []OpenOrder
	if err := json.Unmarshal(body, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}
