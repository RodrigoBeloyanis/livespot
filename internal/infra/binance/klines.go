package binance

import (
	"encoding/json"
	"fmt"
)

type Kline struct {
	OpenTime int64
	Open     string
	High     string
	Low      string
	Close    string
	Volume   string
}

func ParseKlines(body []byte) ([]Kline, error) {
	var raw [][]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]Kline, 0, len(raw))
	for _, row := range raw {
		if len(row) < 6 {
			return nil, fmt.Errorf("kline row invalid")
		}
		openTime, ok := row[0].(float64)
		if !ok {
			return nil, fmt.Errorf("kline open time invalid")
		}
		open, ok := row[1].(string)
		if !ok {
			return nil, fmt.Errorf("kline open invalid")
		}
		high, ok := row[2].(string)
		if !ok {
			return nil, fmt.Errorf("kline high invalid")
		}
		low, ok := row[3].(string)
		if !ok {
			return nil, fmt.Errorf("kline low invalid")
		}
		closeVal, ok := row[4].(string)
		if !ok {
			return nil, fmt.Errorf("kline close invalid")
		}
		volume, ok := row[5].(string)
		if !ok {
			return nil, fmt.Errorf("kline volume invalid")
		}
		out = append(out, Kline{
			OpenTime: int64(openTime),
			Open:     open,
			High:     high,
			Low:      low,
			Close:    closeVal,
			Volume:   volume,
		})
	}
	return out, nil
}

